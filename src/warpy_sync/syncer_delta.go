package warpy_sync

import (
	"context"
	"encoding/hex"
	"errors"
	"strings"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type SyncerDelta struct {
	*task.Task
	monitor                  monitoring.Monitor
	input                    chan *BlockInfoPayload
	Output                   chan *LastSyncedBlockPayload
	OutputInteractionPayload chan *[]InteractionPayload
}

// This task receives block info in the input channel, iterate through all of the block's transactions in order to check if any of it contains
// Redstone data and if so - writes an interaction to Warpy. It emits block height and block hash in the Output channel
func NewSyncerDelta(config *config.Config) (self *SyncerDelta) {
	self = new(SyncerDelta)

	self.Output = make(chan *LastSyncedBlockPayload)
	self.OutputInteractionPayload = make(chan *[]InteractionPayload)

	self.Task = task.NewTask(config, "syncer").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.WarpySyncer.SyncerDeltaNumWorkers, config.WarpySyncer.SyncerDeltaWorkerQueueSize)

	return
}

func (self *SyncerDelta) WithInputChannel(v chan *BlockInfoPayload) *SyncerDelta {
	self.input = v
	return self
}

func (self *SyncerDelta) WithMonitor(monitor monitoring.Monitor) *SyncerDelta {
	self.monitor = monitor
	return self
}

func (self *SyncerDelta) run() (err error) {
	for block := range self.input {
		self.Log.WithField("height", block.Height).Trace("Checking transactions for block")
		var wg sync.WaitGroup
		wg.Add(len(block.Transactions))

		for _, tx := range block.Transactions {
			tx := tx
			block := block
			self.SubmitToWorker(func() {
				err := self.checkTxAndWriteInteraction(tx, block)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.Height).
						Error("Could not process transaction")
					self.monitor.GetReport().WarpySyncer.Errors.SyncerDeltaProcessTxPermanentError.Inc()
					goto end
				}

				self.monitor.GetReport().WarpySyncer.State.SyncerDeltaTxsProcessed.Inc()

			end:
				wg.Done()
			})
		}

		wg.Wait()

		select {
		case <-self.Ctx.Done():
			return nil
		case self.Output <- &LastSyncedBlockPayload{
			Height:    block.Height,
			Hash:      block.Hash,
			Timestamp: block.Timestamp,
		}:
		}

		self.monitor.GetReport().WarpySyncer.State.SyncerDeltaBlocksProcessed.Inc()
	}
	return
}

func (self *SyncerDelta) checkTxAndWriteInteraction(tx *types.Transaction, block *BlockInfoPayload) (err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		// Retries infinitely until success
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.WarpySyncer.SyncerDeltaBackoffInterval).
		WithAcceptableDuration(self.Config.WarpySyncer.SyncerDeltaBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}

			self.monitor.GetReport().WarpySyncer.Errors.SyncerDeltaCheckTxFailures.Inc()
			self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.Height).
				Warn("Could not process transaction, retrying...")
			return err
		}).
		Run(func() error {
			txContainsRedstoneData := self.checkTxForData(tx, self.Config.WarpySyncer.SyncerDeltaRedstoneData, self.Ctx)
			if txContainsRedstoneData {
				self.Log.WithField("txId", tx.Hash()).WithField("height", block.Height).Info("Found new Redstone tx")
				sender, err := eth.GetTxSenderHash(tx)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).Warn("Could not retrieve tx sender")
					return err
				}

				self.OutputInteractionPayload <- &[]InteractionPayload{{
					FromAddress: sender,
					Points:      self.Config.WarpySyncer.SyncerDeltaInteractionPoints,
				}}
			}

			return err
		})

	return
}

func (self *SyncerDelta) checkTxForData(tx *types.Transaction, data string, ctx context.Context) (txContainsData bool) {
	txContainsData = false
	encodedString := hex.EncodeToString(tx.Data())
	if strings.Contains(encodedString, data) {
		txContainsData = true
	}
	return
}
