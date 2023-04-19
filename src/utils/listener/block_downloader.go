package listener

import (
	"bytes"
	"context"
	"errors"
	"math"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"
	"time"

	"github.com/cenkalti/backoff/v4"
	"gorm.io/gorm"
)

// Task that periodically checks for new arweave network info.
// Optionally waits for a number of required confirmation blocks before emitting the info
type BlockDownloader struct {
	*task.Task

	// Runtime configuration
	startHeight            uint64
	previousBlockIndepHash arweave.Base64String

	// Optionally synchonization can be stopped at a certain height
	stopBlockHeight uint64

	client  *arweave.Client
	monitor monitoring.Monitor

	input  chan *arweave.NetworkInfo
	Output chan *arweave.Block

	// Parameters
	maxElapsedTime time.Duration
	maxInterval    time.Duration
}

// Using Arweave client periodically checks for blocks of transactions
func NewBlockDownloader(config *config.Config) (self *BlockDownloader) {
	self = new(BlockDownloader)

	// By default range allows all possible heights
	self.startHeight = 0
	self.stopBlockHeight = math.MaxUint64

	self.Output = make(chan *arweave.Block)

	self.Task = task.NewTask(config, "block-downloader").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *BlockDownloader) WithMonitor(monitor monitoring.Monitor) *BlockDownloader {
	self.monitor = monitor
	return self
}

func (self *BlockDownloader) WithClient(client *arweave.Client) *BlockDownloader {
	self.client = client
	return self
}

func (self *BlockDownloader) WithBackoff(maxElapsedTime, maxInterval time.Duration) *BlockDownloader {
	self.maxElapsedTime = maxElapsedTime
	self.maxInterval = maxInterval
	return self
}

func (self *BlockDownloader) WithHeightRange(start, stop uint64) *BlockDownloader {
	self.Task = self.Task.WithOnBeforeStart(func() (err error) {
		block, err := self.client.GetBlockByHeight(self.Ctx, int64(start-1))
		if err != nil {
			return err
		}

		if !block.IsValid() {
			return errors.New("Invalid block")
		}

		self.startHeight = uint64(block.Height)
		self.previousBlockIndepHash = block.IndepHash
		self.stopBlockHeight = stop

		return nil
	})
	return self
}

func (self *BlockDownloader) WithInitStartHeight(db *gorm.DB, component model.SyncedComponent) *BlockDownloader {
	self.Task = self.Task.WithOnBeforeStart(func() (err error) {
		// Get the last storeserverd block height from the database
		var state model.State
		err = db.WithContext(self.Ctx).Find(&state, component).Error
		if err != nil {
			self.Log.WithError(err).Error("Failed to get last transaction block height")
			return
		}

		self.startHeight = state.FinishedBlockHeight
		self.previousBlockIndepHash = state.FinishedBlockHash

		return nil
	})
	return self
}

func (self *BlockDownloader) WithInputChannel(v chan *arweave.NetworkInfo) *BlockDownloader {
	self.input = v
	return self
}

// Listens for changed height and downloads the missing blocks
func (self *BlockDownloader) run() error {
	lastSyncedHeight := self.startHeight
	lastProcessedBlockHash := self.previousBlockIndepHash

	// Listen for new blocks (blocks)
	// Finishes when Listener is stopping
	for networkInfo := range self.input {

		self.Log.
			WithField("last", lastSyncedHeight).
			WithField("new", networkInfo.Height).
			WithField("numNewBlocks", uint64(networkInfo.Height)-lastSyncedHeight).
			Debug("Discovered new blocks")

		// Download transactions from
		for height := lastSyncedHeight + 1; height <= uint64(networkInfo.Height) && height >= self.startHeight && height <= self.stopBlockHeight; height++ {
			self.monitor.GetReport().BlockDownloader.State.CurrentHeight.Store(int64(height))

			self.Log.WithField("height", height).Trace("Downloading block")

			var block *arweave.Block
			err := task.NewRetry().
				WithContext(self.Ctx).
				WithMaxElapsedTime(self.maxElapsedTime).
				WithMaxInterval(self.maxInterval).
				WithAcceptableDuration(self.maxInterval * 2).
				WithOnError(func(err error, isDurationAcceptable bool) error {
					self.Log.WithError(err).WithField("height", height).Error("Failed to download block, retrying...")

					if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
						// Stopping
						return backoff.Permanent(err)
					}
					self.monitor.GetReport().BlockDownloader.Errors.BlockDownloadErrors.Inc()

					if !isDurationAcceptable {
						// This will completly reset the HTTP client and possibly help in solving the problem
						self.client.Reset()
					}

					return err
				}).
				Run(func() (err error) {
					block, err = self.client.GetBlockByHeight(self.Ctx, int64(height))
					if err != nil {
						return err
					}

					if len(lastProcessedBlockHash) > 0 &&
						!bytes.Equal(lastProcessedBlockHash, block.PreviousBlock) {
						self.Log.WithField("height", height).
							WithField("last_processed_block_hash", lastProcessedBlockHash).
							WithField("previous_block", block.PreviousBlock).
							Error("Previous block hash isn't valid")

						self.monitor.GetReport().BlockDownloader.Errors.BlockValidationErrors.Inc()

						// TODO: Try downloading with another peer
						// TODO: Log malicious peer
						err = errors.New("previous block hash isn't valid")
						return
					}

					if !block.IsValid() {
						self.Log.WithField("height", height).Error("Block hash isn't valid")
						self.monitor.GetReport().BlockDownloader.Errors.BlockValidationErrors.Inc()
						err = errors.New("block isn't valid")
						return
					}

					return
				})

			if err != nil {
				self.Log.WithError(err).WithField("height", height).Error("Failed to download block, stop retrying")
				return err
			}

			self.Log.
				WithField("height", height).
				WithField("len", len(block.Txs)).
				Debug("Downloaded block")

			// Blocks until something is ready to receive
			self.Output <- block

			// Prepare for the next block
			lastSyncedHeight = uint64(block.Height)
			lastProcessedBlockHash = block.IndepHash

			// Update monitoring
			self.monitor.GetReport().BlockDownloader.State.CurrentHeight.Store(block.Height)
		}
	}

	return nil
}
