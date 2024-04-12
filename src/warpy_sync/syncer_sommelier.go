package warpy_sync

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"gorm.io/gorm"
)

type SyncerSommelier struct {
	*task.Task
	monitor                  monitoring.Monitor
	db                       *gorm.DB
	input                    chan *BlockInfoPayload
	Output                   chan *LastSyncedBlockPayload
	OutputTransactionPayload chan *SommelierTransactionPayload

	contractAbi *abi.ABI
}

// This task receives block info in the input channel, iterate through all of the block's transactions in order to check if any contains
// Sommelier transaction and saves transaction in the db
func NewSyncerSommelier(config *config.Config) (self *SyncerSommelier) {
	self = new(SyncerSommelier)

	self.Output = make(chan *LastSyncedBlockPayload)

	self.OutputTransactionPayload = make(chan *SommelierTransactionPayload)

	self.Task = task.NewTask(config, "syncer").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.WarpySyncer.SyncerSommelierNumWorkers, config.WarpySyncer.SyncerSommelierWorkerQueueSize)

	return
}

func (self *SyncerSommelier) WithInputChannel(v chan *BlockInfoPayload) *SyncerSommelier {
	self.input = v
	return self
}

func (self *SyncerSommelier) WithMonitor(monitor monitoring.Monitor) *SyncerSommelier {
	self.monitor = monitor
	return self
}

func (self *SyncerSommelier) WithDb(v *gorm.DB) *SyncerSommelier {
	self.db = v
	return self
}

func (self *SyncerSommelier) WithContractAbi(contractAbi *abi.ABI) *SyncerSommelier {
	self.contractAbi = contractAbi
	return self
}

func (self *SyncerSommelier) run() (err error) {
	for block := range self.input {
		self.Log.WithField("height", block.Height).Trace("Checking transactions for block")
		var wg sync.WaitGroup
		wg.Add(len(block.Transactions))

		for _, tx := range block.Transactions {
			tx := tx
			block := block
			self.SubmitToWorker(func() {
				err := self.checkTx(tx, block)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.Height).
						Error("Could not process transaction")
					self.monitor.GetReport().WarpySyncer.Errors.SyncerSommelierProcessTxPermanentError.Inc()
					goto end
				}

				self.monitor.GetReport().WarpySyncer.State.SyncerSommelierTxsProcessed.Inc()

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

		self.monitor.GetReport().WarpySyncer.State.SyncerSommelierBlocksProcessed.Inc()
	}
	return
}

func (self *SyncerSommelier) checkTx(tx *types.Transaction, block *BlockInfoPayload) (err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		// Retries infinitely until success
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.WarpySyncer.SyncerSommelierBackoffInterval).
		WithAcceptableDuration(self.Config.WarpySyncer.SyncerSommelierBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}

			self.monitor.GetReport().WarpySyncer.Errors.SyncerSommelierCheckTxFailures.Inc()
			self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.Height).
				Warn("Could not process transaction, retrying...")
			return err
		}).
		Run(func() error {
			if tx.To() != nil && tx.To().String() == self.Config.WarpySyncer.SyncerSommelierContractId {
				self.Log.WithField("tx_id", tx.Hash()).Info("Found new Sommelier transaction")
				method, inputsMap, err := eth.DecodeTransactionInputData(self.contractAbi, tx.Data())
				if err != nil {
					self.Log.WithError(err).Error("Could not decode transaction input data")
					return nil
				}

				if slices.Contains(self.Config.WarpySyncer.SyncerSommelierFunctions, method.Name) {
					parsedInputsMap, err := json.Marshal(inputsMap)
					if err != nil {
						self.Log.WithError(err).Error("Could not parse transaction input")
						return err
					}

					self.Log.WithField("method_name", method.Name).WithField("inputs_map", string(parsedInputsMap)).
						Info("New transaction decoded")

					from, err := eth.GetTxSenderHash(tx)
					if err != nil {
						self.Log.WithError(err).Error("Could not get transaction sender")
						return err
					}

					select {
					case <-self.Ctx.Done():
						return nil
					case self.OutputTransactionPayload <- &SommelierTransactionPayload{
						Transaction: tx,
						FromAddress: from,
						Block:       block,
						Method:      method,
						ParsedInput: parsedInputsMap,
						Input:       inputsMap,
					}:
					}
				}
			}

			return err
		})

	return
}
