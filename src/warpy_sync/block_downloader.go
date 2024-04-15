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
			block, err := self.downloadBlock(height)
			if err != nil {
				self.Log.WithError(err).WithField("height", height).Error("Failed to download block")
				goto end
			}

			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &BlockInfoPayload{
				Transactions: block.Transactions(),
				Height:       block.Number().Uint64(),
				Hash:         block.Hash().String(),
				Timestamp:    block.Time(),
			}:
				if self.pollerCron && height == nextPollBlockHeight {
					self.OutputPollTxs <- block.Number().Uint64()

					self.nextPollBlockHeight = self.calculateNextFullBlockHeight(block.Number().Int64(), int64(block.Time()))
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

func (self *BlockDownloader) downloadBlock(height int64) (block *types.Block, err error) {
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
			block, err = self.getBlockInfo(height)
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

func (self *BlockDownloader) getBlockInfo(blockNumber int64) (blockInfo *types.Block, err error) {
	blockInfo, err = self.ethClient.BlockByNumber(context.Background(), big.NewInt(blockNumber))
	if err != nil {
		return
	}
	return
}

func (self *BlockDownloader) calculateNextFullBlockHeight(lastSyncedBlockHeight int64, lastSyncedBlockTimestamp int64) int64 {
	nextPollBlockTimestamp := (lastSyncedBlockTimestamp - (lastSyncedBlockTimestamp % self.Config.WarpySyncer.BlockDownloaderPollerInterval)) + self.Config.WarpySyncer.BlockDownloaderPollerInterval
	timestampDiff := nextPollBlockTimestamp - lastSyncedBlockTimestamp
	blocksDiff := math.Round(float64(timestampDiff) / float64(0.26))
	return lastSyncedBlockHeight + int64(blocksDiff)
}
