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

type SyncerDeposit struct {
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
func NewSyncerDeposit(config *config.Config) (self *SyncerDeposit) {
	self = new(SyncerDeposit)

	self.Output = make(chan *LastSyncedBlockPayload)

	self.OutputTransactionPayload = make(chan *SommelierTransactionPayload)

	self.Task = task.NewTask(config, "syncer_deposit").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.WarpySyncer.SyncerDepositNumWorkers, config.WarpySyncer.SyncerDepositWorkerQueueSize)

	return
}

func (self *SyncerDeposit) WithInputChannel(v chan *BlockInfoPayload) *SyncerDeposit {
	self.input = v
	return self
}

func (self *SyncerDeposit) WithMonitor(monitor monitoring.Monitor) *SyncerDeposit {
	self.monitor = monitor
	return self
}

func (self *SyncerDeposit) WithDb(v *gorm.DB) *SyncerDeposit {
	self.db = v
	return self
}

func (self *SyncerDeposit) WithContractAbi(contractAbi *abi.ABI) *SyncerDeposit {
	self.contractAbi = contractAbi
	return self
}

func (self *SyncerDeposit) run() (err error) {
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
					self.monitor.GetReport().WarpySyncer.Errors.SyncerDepositProcessTxPermanentError.Inc()
					goto end
				}

				self.monitor.GetReport().WarpySyncer.State.SyncerDepositTxsProcessed.Inc()

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

		self.monitor.GetReport().WarpySyncer.State.SyncerDepositBlocksProcessed.Inc()
	}
	return
}

func (self *SyncerDeposit) checkTx(tx *types.Transaction, block *BlockInfoPayload) (err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		// Retries infinitely until success
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.WarpySyncer.SyncerDepositBackoffInterval).
		WithAcceptableDuration(self.Config.WarpySyncer.SyncerDepositBackoffInterval * 2).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				return backoff.Permanent(err)
			}

			self.monitor.GetReport().WarpySyncer.Errors.SyncerDepositCheckTxFailures.Inc()
			self.Log.WithError(err).WithField("txId", tx.Hash()).WithField("height", block.Height).
				Warn("Could not process transaction, retrying...")
			return err
		}).
		Run(func() error {
			if tx.To() != nil && tx.To().String() == self.Config.WarpySyncer.SyncerDepositContractId {
				self.Log.WithField("tx_id", tx.Hash()).Info("Found new deposit transaction")
				method, inputsMap, err := eth.DecodeTransactionInputData(self.contractAbi, tx.Data())
				if err != nil {
					self.Log.WithError(err).Error("Could not decode transaction input data")
					return nil
				}

				if slices.Contains(self.Config.WarpySyncer.SyncerDepositFunctions, method.Name) {
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
