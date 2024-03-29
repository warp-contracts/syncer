package relay

import (
	"bytes"
	"context"
	"errors"
	"sort"

	"github.com/cenkalti/backoff"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
)

// Task for filling in the arweave blocks in Payload
type OneBlockDownloader struct {
	*task.Task

	client  *arweave.Client
	monitor monitoring.Monitor

	input  chan *Payload
	Output chan *Payload

	lastBlockHeight int64
	lastBlockHash   arweave.Base64String
}

// Using Arweave client periodically checks for blocks of transactions
func NewOneBlockDownloader(config *config.Config) (self *OneBlockDownloader) {
	self = new(OneBlockDownloader)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "one-block-downloader").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *OneBlockDownloader) WithMonitor(monitor monitoring.Monitor) *OneBlockDownloader {
	self.monitor = monitor
	return self
}

func (self *OneBlockDownloader) WithClient(client *arweave.Client) *OneBlockDownloader {
	self.client = client
	return self
}

func (self *OneBlockDownloader) WithInputChannel(v chan *Payload) *OneBlockDownloader {
	self.input = v
	return self
}

func (self *OneBlockDownloader) downloadBlock(arweaveBlock *ArweaveBlock) (block *arweave.Block, err error) {
	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.Relayer.ArweaveBlockDownloadTimeout)
	defer cancel()

	// Download block
	block, resp, err := self.client.GetBlockByHash(ctx, arweaveBlock.Message.BlockInfo.Hash)
	if err != nil {
		self.Log.
			WithError(err).
			WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
			Error("Failed to download block")
		return
	}

	// Check if this is really the desired block
	if block.IndepHash.Base64() != arweaveBlock.Message.BlockInfo.Hash {
		self.Log.
			WithField("requested_height", arweaveBlock.Message.BlockInfo.Height).
			WithField("requested_hash", arweaveBlock.Message.BlockInfo.Hash).
			WithField("received_hash", block.IndepHash.Base64()).
			Error("Block hash doesn't match")
		self.monitor.GetReport().BlockDownloader.Errors.BlockValidationErrors.Inc()
		err = errors.New("block hash isn't what we expected")
		return
	}

	if block.Height != int64(arweaveBlock.Message.BlockInfo.Height) {
		self.Log.
			WithField("requested_height", arweaveBlock.Message.BlockInfo.Height).
			WithField("requested_hash", arweaveBlock.Message.BlockInfo.Hash).
			WithField("received_height", block.Height).
			Error("Block height doesn't match")
		self.monitor.GetReport().BlockDownloader.Errors.BlockValidationErrors.Inc()
		err = errors.New("block height isn't what we expected")
		return
	}

	if len(self.lastBlockHash) > 0 &&
		!bytes.Equal(self.lastBlockHash, block.PreviousBlock) {
		self.Log.
			WithField("height", arweaveBlock.Message.BlockInfo.Height).
			WithField("age", resp.Header().Get("Age")).
			WithField("x-trace", resp.Header().Get("X-Trace")).
			WithField("last_block_hash", self.lastBlockHash.Base64()).
			WithField("previous_block", block.PreviousBlock.Base64()).
			Warn("Previous block hash isn't valid")
		err = errors.New("previous block hash isn't valid")
		return
	}

	// Check if all transactions in message are in the block
	for _, msgTx := range arweaveBlock.Message.Transactions {
		var msgTxBytes arweave.Base64String
		err = msgTxBytes.Decode(msgTx.Transaction.Id)
		if err != nil {
			self.Log.WithError(err).Error("Failed to decode arweave transaction id from sequencer message")
			return
		}
		found := false
		for _, tx := range block.Txs {
			if bytes.Equal(tx, msgTxBytes) {
				found = true
				break
			}
		}

		if !found {
			err = errors.New("transaction from sequencer message not found in arweave block")
		}
	}

	if !block.IsValid() {
		self.Log.
			WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
			WithField("age", resp.Header().Get("Age")).
			WithField("x-trace", resp.Header().Get("X-Trace")).
			Error("Block hash isn't valid")
		self.monitor.GetReport().BlockDownloader.Errors.BlockValidationErrors.Inc()
		err = errors.New("block isn't valid")
		return
	}

	return
}

func (self *OneBlockDownloader) download(arweaveBlock *ArweaveBlock) (out *arweave.Block, err error) {
	self.Log.
		WithField("last_height", self.lastBlockHeight).
		WithField("last_hash", self.lastBlockHash.Base64()).
		WithField("height", arweaveBlock.Message.BlockInfo.Height).
		WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
		Debug("Downloading block...")

	defer self.Log.WithField("height", arweaveBlock.Message.BlockInfo.Height).
		WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
		Debug("...Downloaded block")

	err = task.NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(self.Config.Relayer.ArweaveBlockDownloadMaxElapsedTime).
		WithMaxInterval(self.Config.Relayer.ArweaveBlockDownloadMaxInterval).
		WithAcceptableDuration(self.Config.Relayer.ArweaveBlockDownloadTimeout * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				// Stopping
				return backoff.Permanent(err)
			}
			self.Log.WithError(err).
				WithField("height", arweaveBlock.Message.BlockInfo.Height).
				WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
				Error("Failed to download block, retrying...")

			self.monitor.GetReport().BlockDownloader.Errors.BlockDownloadErrors.Inc()

			if !isDurationAcceptable {
				// This will completly reset the HTTP client and possibly help in solving the problem
				self.client.Reset()
			}

			return err
		}).
		Run(func() (err error) {
			out, err = self.downloadBlock(arweaveBlock)
			return
		})

	if err != nil {
		self.Log.WithError(err).
			WithField("height", arweaveBlock.Message.BlockInfo.Height).
			WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
			Error("Failed to download block, stop retrying")
		return
	}

	return
}

// Listens for changed height and downloads the missing blocks
func (self *OneBlockDownloader) run() (err error) {
	for payload := range self.input {
		// Make sure blocks are in order of height
		sort.Slice(payload.ArweaveBlocks, func(i, j int) bool {
			return payload.ArweaveBlocks[i].Message.BlockInfo.Height < payload.ArweaveBlocks[j].Message.BlockInfo.Height
		})

		// Download blocks one by one
		for i, arweaveBlock := range payload.ArweaveBlocks {
			if arweaveBlock == nil {
				// TODO: This was a for debbuging, remove after confirming it doesn't happen anymore
				self.Log.WithField("idx", i).Warn("Arweave block is nil, skipping")
				continue
			}
			payload.ArweaveBlocks[i].Block, err = self.download(arweaveBlock)
			if err != nil {
				if self.IsStopping.Load() {
					// Neglect the block, we're stopping anyway
					return nil
				}

				self.Log.WithError(err).
					WithField("sequencer_height", payload.SequencerBlockHeight).
					WithField("arweave_hash", arweaveBlock.Message).
					Error("Failed to download block, stop retrying")

				// Stop everything
				// We can't neglect a missing block
				panic(err)
			}

			// Prepare for the next block
			self.lastBlockHeight = payload.ArweaveBlocks[i].Block.Height
			self.lastBlockHash = payload.ArweaveBlocks[i].Block.IndepHash

			// Update monitoring
			self.monitor.GetReport().BlockDownloader.State.CurrentHeight.Store(arweaveBlock.Block.Height)
		}

		// Arweave blocks filled
		self.Output <- payload
	}

	return nil
}
