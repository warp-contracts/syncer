package redstone_tx_sync

import (
	"context"
	"errors"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/redstone_tx_sync"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Syncer struct {
	*task.Task
	monitor monitoring.Monitor
	client  *redstone_tx_sync.Client
	input   chan *Payload
}

func NewSyncer(config *config.Config) (self *Syncer) {
	self = new(Syncer)

	self.Task = task.NewTask(config, "syncer").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.RedstoneTxSyncer.SyncerNumWorkers, config.RedstoneTxSyncer.SyncerWorkerQueueSize)

	return
}

func (self *Syncer) WithClient(redstoneSyncerClient *redstone_tx_sync.Client) *Syncer {
	self.client = redstoneSyncerClient
	return self
}

func (self *Syncer) WithInputChannel(v chan *Payload) *Syncer {
	self.input = v
	return self
}

func (self *Syncer) WithMonitor(monitor monitoring.Monitor) *Syncer {
	self.monitor = monitor
	return self
}

func (self *Syncer) run() (err error) {
	for block := range self.input {
		self.Log.WithField("height", block.BlockHeight).Debug("Checking transactions for block")
		var wg sync.WaitGroup
		wg.Add(len(block.Transactions))
		var mtx sync.Mutex

		for _, tx := range block.Transactions {
			tx := tx
			self.SubmitToWorker(func() {
				mtx.Lock()
				defer mtx.Unlock()
				err = self.checkTxAndWriteInteraction(tx, block)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.BlockHeight).
						Error("Could not process transaction")
					return
				}

				self.monitor.GetReport().RedstoneTxSyncer.State.SyncerTxsProcessed.Inc()
				wg.Done()
			})
		}

		wg.Wait()

		err = self.updateSyncState(block.BlockHeight, block.BlockHash)
		if err != nil {
			self.Log.WithError(err).Error("Could not update sync_state")
			return err
		}
		self.Log.WithField("blockHeight", block.BlockHeight).Debug("sync_state updated")
		self.monitor.GetReport().RedstoneTxSyncer.State.SyncerBlocksProcessed.Inc()
	}

	return
}

func (self *Syncer) checkTxAndWriteInteraction(tx *types.Transaction, block *Payload) (err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.RedstoneTxSyncer.SyncerBackoffInterval).
		WithAcceptableDuration(self.Config.RedstoneTxSyncer.SyncerBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}

			self.monitor.GetReport().RedstoneTxSyncer.Errors.SyncerWriteInteractionsFailures.Inc()
			self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.BlockHeight).
				Warn("Could not process transaction, retrying...")
			return err
		}).
		Run(func() error {
			txContainsRedstoneData := self.client.CheckTxForData(tx, self.Config.RedstoneTxSyncer.SyncerRedstoneData, self.Ctx)
			if txContainsRedstoneData {
				self.Log.WithField("txId", tx.Hash()).WithField("height", block.BlockHeight).Info("Found new Redstone tx")
				sender, err := self.client.GetTxSenderHash(tx)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).Warn("Could not retrieve tx sender")
					return err
				}
				input := Input{
					Function: "addPointsCsv",
					Points:   self.Config.RedstoneTxSyncer.SyncerInteractionPoints,
					AdminId:  self.Config.RedstoneTxSyncer.SyncerInteractionAdminId,
					Members:  []Member{{Id: sender, Roles: []string{}}},
					NoBoost:  true,
				}
				self.Log.WithField("txId", tx.Hash()).Debug("Writing interaction to Warpy...")
				interactionId, err := self.client.WriteInteractionToWarpy(
					self.Ctx, tx, self.Config.RedstoneTxSyncer.SyncerSigner, input, self.Config.RedstoneTxSyncer.SyncerContractId)
				if err != nil {
					return err
				}
				self.Log.WithField("interactionId", interactionId).Info("Interaction sent to Warpy")
				self.monitor.GetReport().RedstoneTxSyncer.State.SyncerInteractionsToWarpy.Inc()
			}

			return err
		})

	return
}

func (self *Syncer) updateSyncState(blockHeight int64, blockHash string) (err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.RedstoneTxSyncer.SyncerBackoffInterval).
		WithAcceptableDuration(self.Config.RedstoneTxSyncer.SyncerBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}

			self.monitor.GetReport().RedstoneTxSyncer.Errors.SyncerUpdateSyncStateFailures.Inc()
			self.Log.WithError(err).Warn("Could not update sync_state, retrying...")
			return err
		}).
		Run(func() error {
			err = self.client.UpdateSyncState(self.Ctx, blockHeight, blockHash)
			return err
		})
	return
}
