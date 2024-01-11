package redstone_tx_sync

import (
	"context"
	"errors"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	redstone_tx_sync "github.com/warp-contracts/syncer/src/utils/redstone_tx_sync"
	"github.com/warp-contracts/syncer/src/utils/task"
	"gorm.io/gorm"
)

type BlockDownloader struct {
	*task.Task
	client                *redstone_tx_sync.Client
	lastSyncedBlockHeight int64
	Output                chan *Payload
	monitor               monitoring.Monitor
}

func NewBlockDownloader(config *config.Config) (self *BlockDownloader) {
	self = new(BlockDownloader)
	self.Output = make(chan *Payload, config.RedstoneTxSyncer.BlockDownloaderBufferLength)

	self.Task = task.NewTask(config, "block-downloader").
		WithPeriodicSubtaskFunc(config.RedstoneTxSyncer.BlockDownloaderInterval, self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *BlockDownloader) WithClient(client *redstone_tx_sync.Client) *BlockDownloader {
	self.client = client
	return self
}

func (self *BlockDownloader) WithMonitor(monitor monitoring.Monitor) *BlockDownloader {
	self.monitor = monitor
	return self
}

func (self *BlockDownloader) WithInitStartBlockHeight(db *gorm.DB) *BlockDownloader {
	self.Task = self.Task.WithOnBeforeStart(func() (err error) {
		lastSyncedBlockHeight, err := self.client.GetLastSyncedBlockHeight(self.Ctx)
		if err != nil {
			self.Log.WithError(err).Error("Failed to get last synced block height")
		}

		self.lastSyncedBlockHeight = lastSyncedBlockHeight
		return nil
	})
	return self
}

func (self *BlockDownloader) run() (err error) {
	currentBlockHeight, _, err := self.client.GetCurrentBlockHeight()
	if err != nil {
		self.Log.WithError(err).Error("Could not get current block height")
		return err
	}

	self.Log.
		WithField("lastSyncedBlockHeight", self.lastSyncedBlockHeight).
		WithField("currentHeight", currentBlockHeight).
		WithField("newBlocks", currentBlockHeight-self.lastSyncedBlockHeight).
		Info("Discovered new blocks")

	for height := self.lastSyncedBlockHeight + 1; height <= currentBlockHeight &&
		height <= (self.lastSyncedBlockHeight+int64(self.Config.RedstoneTxSyncer.BlockDownloaderMaxRange)); height++ {

		block, err := self.downloadBlock(height)
		if err != nil {
			self.Log.WithError(err).WithField("height", height).Error("Failed to download block")
			return err
		}

		self.Log.WithField("height", height).Debug("New block downloaded")

		payload := &Payload{
			Transactions: block.Transactions(),
			BlockHeight:  block.Number().Int64(),
			BlockHash:    block.Hash().String(),
		}

		self.Output <- payload

		self.lastSyncedBlockHeight = payload.BlockHeight
		self.monitor.GetReport().RedstoneTxSyncer.State.BlockDownloaderCurrentHeight.Store(payload.BlockHeight)
	}

	return
}

func (self *BlockDownloader) downloadBlock(height int64) (block *types.Block, err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.RedstoneTxSyncer.BlockDownloaderBackoffInterval).
		WithAcceptableDuration(self.Config.RedstoneTxSyncer.BlockDownloaderBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}
			self.monitor.GetReport().RedstoneTxSyncer.Errors.BlockDownloaderFailures.Inc()
			self.Log.WithError(err).WithField("height", height).Warn("Failed to download block, retrying...")
			return err
		}).
		Run(func() (err error) {
			block, err = self.client.GetBlockInfo(height)
			if err != nil {
				return err
			}

			return
		})

	return
}
