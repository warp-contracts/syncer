package warpy_sync

import (
	"context"
	"errors"
	"math"
	"math/big"
	"runtime"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"gorm.io/gorm"
)

type BlockDownloader struct {
	*task.Task
	lastSyncedBlockHeight int64
	pollerCron            bool
	nextPollBlockHeight   int64
	Output                chan *BlockInfoPayload
	OutputPollTxs         chan uint64
	monitor               monitoring.Monitor
	ethClient             *ethclient.Client
}

// When started, this task checks database for the latest block synced and store the result in lastSyncedBlockHeight field
// Then, compare lastSyncedBlockHeight to the current external network's block height and loads info for all of the missing
// blocks for the indicated range
// Lastly, it emits to the Output channel block's height, block's hash and all transactions that this block contains and
// update lastSyncedBlockHeight to the last processed block
func NewBlockDownloader(config *config.Config) (self *BlockDownloader) {
	self = new(BlockDownloader)
	self.Output = make(chan *BlockInfoPayload, config.WarpySyncer.BlockDownloaderChannelSize)
	self.OutputPollTxs = make(chan uint64, config.WarpySyncer.BlockDownloaderChannelSize)

	self.Task = task.NewTask(config, "block-downloader").
		WithPeriodicSubtaskFunc(config.WarpySyncer.BlockDownloaderInterval, self.run).
		WithWorkerPool(runtime.NumCPU(), config.WarpySyncer.BlockDownloaderMaxQueueSize).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *BlockDownloader) WithMonitor(monitor monitoring.Monitor) *BlockDownloader {
	self.monitor = monitor
	return self
}

func (self *BlockDownloader) WithPollerCron() *BlockDownloader {
	self.pollerCron = true
	return self
}

func (self *BlockDownloader) WithEthClient(ethClient *ethclient.Client) *BlockDownloader {
	self.ethClient = ethClient
	return self
}

func (self *BlockDownloader) WithInitStartBlockHeight(db *gorm.DB, syncedComponent model.SyncedComponent) *BlockDownloader {
	self.Task = self.Task.WithOnBeforeStart(func() (err error) {
		var LastSyncedBlock struct {
			FinishedBlockHeight    int64
			FinishedBlockTimestamp int64
		}

		err = db.WithContext(self.Ctx).
			Raw(`SELECT finished_block_height, finished_block_timestamp
			FROM sync_state
			WHERE name = ?;`, syncedComponent).
			Scan(&LastSyncedBlock).Error

		if err != nil {
			self.Log.WithError(err).Error("Failed to get last synced block height")
			return
		}

		self.lastSyncedBlockHeight = LastSyncedBlock.FinishedBlockHeight
		if self.pollerCron {
			self.nextPollBlockHeight = self.calculateNextFullBlockHeight(self.lastSyncedBlockHeight, LastSyncedBlock.FinishedBlockTimestamp)
			self.Log.WithField("block_height", self.nextPollBlockHeight).Debug("Initial full block height has been set")
		}
		return nil
	})
	return self
}

func (self *BlockDownloader) run() (err error) {
	currentBlockHeight, err := self.getCurrentBlockHeight(self.Ctx)
	if err != nil {
		self.Log.WithError(err).Error("Could not get current block height")
		return err
	}

	if currentBlockHeight == self.lastSyncedBlockHeight {
		self.Log.Debug("No new blocks found, exiting")
		return
	}

	self.Log.
		WithField("lastSyncedBlockHeight", self.lastSyncedBlockHeight).
		WithField("currentHeight", currentBlockHeight).
		WithField("newBlocks", currentBlockHeight-self.lastSyncedBlockHeight).
		Info("Discovered new blocks")

	for ok := true; ok; ok = self.lastSyncedBlockHeight < currentBlockHeight-1 {
		batchSize := min(self.Config.WarpySyncer.BlockDownloaderBatchSize, int(currentBlockHeight-self.lastSyncedBlockHeight))

		blocks := make([]int64, batchSize)
		for i := range blocks {
			blocks[i] = int64(i) + self.lastSyncedBlockHeight + 1
		}
		err := self.downloadBlocks(blocks)
		if err != nil {
			return err
		}

		self.lastSyncedBlockHeight = blocks[len(blocks)-1]
		self.Log.WithField("len", len(blocks)).WithField("from", blocks[0]).WithField("to", self.lastSyncedBlockHeight).Info("New blocks downloaded")
		self.monitor.GetReport().WarpySyncer.State.BlockDownloaderCurrentHeight.Store(self.lastSyncedBlockHeight)
	}

	return
}

func (self *BlockDownloader) downloadBlocks(blocks []int64) (err error) {
	var wg sync.WaitGroup
	wg.Add(len(blocks))

	for _, height := range blocks {
		height := height
		nextPollBlockHeight := self.nextPollBlockHeight
		self.SubmitToWorker(func() {
			blockHeight, blockHash, blockTime, blockTxs, err := self.downloadBlockOrHeader(height)
			if err != nil {
				self.Log.WithError(err).WithField("height", height).Error("Failed to download block")
				goto end
			}

			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &BlockInfoPayload{
				Transactions: blockTxs,
				Height:       blockHeight,
				Hash:         blockHash,
				Timestamp:    blockTime,
			}:
				if self.pollerCron && height == nextPollBlockHeight {
					self.OutputPollTxs <- blockHeight

					self.nextPollBlockHeight = int64(blockHeight) + int64(float64(self.Config.WarpySyncer.BlockDownloaderPollerInterval)/self.Config.WarpySyncer.BlockDownloaderBlockTime)
					self.Log.WithField("next_poll_block_height", self.nextPollBlockHeight).Debug("Next poll block height has been set")
				}
			}

		end:
			wg.Done()
		})
	}

	wg.Wait()
	return
}

func (self *BlockDownloader) downloadBlockOrHeader(height int64) (blockHeight uint64, blockHash string, blockTime uint64, blockTxs []*types.Transaction, err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		// Retries infinitely until success
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.WarpySyncer.BlockDownloaderBackoffInterval).
		WithAcceptableDuration(self.Config.WarpySyncer.BlockDownloaderBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}
			self.monitor.GetReport().WarpySyncer.Errors.BlockDownloaderFailures.Inc()
			self.Log.WithError(err).WithField("height", height).Warn("Failed to download block, retrying...")
			return err
		}).
		Run(func() (err error) {
			blockHeight, blockHash, blockTime, blockTxs, err = self.getBlockOrHeaderInfo(height)
			return
		})

	return
}

func (self *BlockDownloader) getCurrentBlockHeight(ctx context.Context) (currentBlockHeight int64, err error) {
	header, err := self.ethClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return
	}

	currentBlockHeight = header.Number.Int64()

	return
}

func (self *BlockDownloader) getBlockOrHeaderInfo(height int64) (blockHeight uint64, blockHash string, blockTime uint64, blockTxs []*types.Transaction, err error) {
	if self.Config.WarpySyncer.BlockDownloaderByHeader {
		header, errHeader := self.ethClient.HeaderByNumber(context.Background(), big.NewInt(height))
		err = errHeader
		if err != nil {
			return
		}
		blockHeight = header.Number.Uint64()
		blockHash = header.Hash().String()
		blockTime = header.Time
		txCount, err := self.ethClient.TransactionCount(context.Background(), header.Hash())
		if err != nil {
			self.Log.WithField("err", err).Warn("could not get transaction count")
		}

		for i := 0; i < int(txCount); i++ {
			tx, err := self.ethClient.TransactionInBlock(context.Background(), header.Hash(), uint(i))
			if err != nil {
				continue
			}

			blockTxs = append(blockTxs, tx)
		}
	} else {
		block, errBlock := self.ethClient.BlockByNumber(context.Background(), big.NewInt(height))
		err = errBlock
		if err != nil {
			return
		}
		blockTxs = block.Transactions()
		blockHeight = block.Number().Uint64()
		blockHash = block.Hash().String()
		blockTime = block.Time()
	}

	return
}

func (self *BlockDownloader) calculateNextFullBlockHeight(lastSyncedBlockHeight int64, lastSyncedBlockTimestamp int64) int64 {
	blockTime, err := self.Config.WarpySyncer.SyncerChain.BlockTime()
	if err != nil {
		self.Log.WithField("chain", self.Config.WarpySyncer.SyncerChain.String()).WithError(err).
			Panic("Cannot calculate next block height")
	}
	nextPollBlockTimestamp := (lastSyncedBlockTimestamp - (lastSyncedBlockTimestamp % self.Config.WarpySyncer.BlockDownloaderPollerInterval)) + self.Config.WarpySyncer.BlockDownloaderPollerInterval
	timestampDiff := nextPollBlockTimestamp - lastSyncedBlockTimestamp
	blocksDiff := math.Round(float64(timestampDiff) / blockTime)
	return lastSyncedBlockHeight + int64(blocksDiff)
}
