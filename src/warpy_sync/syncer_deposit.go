package warpy_sync

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/go-resty/resty/v2"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warpy"
	"gorm.io/gorm"
)

type SyncerDeposit struct {
	*task.Task
	monitor                  monitoring.Monitor
	db                       *gorm.DB
	input                    chan *BlockInfoPayload
	Output                   chan *LastSyncedBlockPayload
	OutputTransactionPayload chan *SommelierTransactionPayload
	contractAbi              map[string]*abi.ABI
	httpClient               *resty.Client
}

// This task receives block info in the input channel, iterate through all of the block's transactions in order to check if any contains
// Sommelier transaction and saves transaction in the db
func NewSyncerDeposit(config *config.Config) (self *SyncerDeposit) {
	self = new(SyncerDeposit)

	self.Output = make(chan *LastSyncedBlockPayload)

	self.OutputTransactionPayload = make(chan *SommelierTransactionPayload)

	self.httpClient = resty.New().
		SetTimeout(config.WarpySyncer.WriterHttpRequestTimeout)

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

func (self *SyncerDeposit) WithContractAbi(contractAbi map[string]*abi.ABI) *SyncerDeposit {
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
			if tx.To() != nil && slices.Contains(self.Config.WarpySyncer.SyncerDepositContractIds, tx.To().String()) {
				self.Log.WithField("tx_id", tx.Hash()).WithField("tx_to", tx.To().String()).Info("Found new on-chain transaction")

				sender, err := eth.GetTxSenderHash(tx)
				if err != nil {
					self.Log.WithError(err).WithField("txId", tx.Hash()).Warn("Could not retrieve tx sender")
					return err
				}

				warpyUser, err := warpy.GetWarpyUserId(self.httpClient, self.Config.WarpySyncer.SyncerDreUrl, sender)
				if err != nil {
					self.Log.WithError(err).WithField("sender", sender).Warn("Could not retrieve user id")
					return err
				}
				if warpyUser == "" {
					self.Log.WithField("sender", sender).Warn("Sender not registered in Warpy")
					return nil
				}

				method, inputsMap, err := eth.DecodeTransactionInputData(self.contractAbi[tx.To().String()], tx.Data())
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

					if (self.Config.WarpySyncer.SyncerChain == eth.Mode) || (self.Config.WarpySyncer.SyncerChain == eth.Manta) {
						contains := self.containsSpecificToken(inputsMap)
						if !contains {
							return nil
						}
					}

					// need to check if deposit is being sent to one of the specific pools
					if self.Config.WarpySyncer.SyncerProtocol == eth.Pendle {
						belongs := self.belongsToMarket(inputsMap, tx.Hash())
						if !belongs {
							return nil
						}
					}

					self.Log.WithField("method_name", method.Name).WithField("inputs_map", string(parsedInputsMap)).
						Info("New transaction decoded")

					select {
					case <-self.Ctx.Done():
						return nil
					case self.OutputTransactionPayload <- &SommelierTransactionPayload{
						Transaction: tx,
						FromAddress: sender,
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

func (self *SyncerDeposit) containsSpecificToken(inputsMap map[string]interface{}) bool {
	tokenName := inputsMap["lToken"]
	tokenNameStr := fmt.Sprintf("%v", tokenName)
	self.Log.WithField("token_name", tokenNameStr).Info("Token set for transfer")

	// TODO: do it properly (i.e. via params, not hardcoded)
	if self.Config.WarpySyncer.SyncerChain == eth.Mode {
		if tokenNameStr != self.Config.WarpySyncer.SyncerDepositToken {
			self.Log.WithField("token_name", tokenNameStr).Warn("Wrong token set for transfer")
			return false
		}
	}

	if self.Config.WarpySyncer.SyncerChain == eth.Manta {
		if tokenNameStr != self.Config.WarpySyncer.SyncerDepositToken {
			self.Log.WithField("token_id", tokenNameStr).Warn("Wrong token set for transfer")
			return false
		}
	}

	return true
}

func (self *SyncerDeposit) belongsToMarket(inputsMap map[string]interface{}, txHash common.Hash) bool {
	market := inputsMap["market"]
	marketStr := fmt.Sprintf("%v", market)

	if !(slices.Contains(self.Config.WarpySyncer.SyncerDepositMarkets, marketStr)) {
		self.Log.WithField("tx_id", txHash).WithField("market", marketStr).Warn("Token deposit does not belong to desired market")
		return false
	}

	self.Log.WithField("tx_id", txHash).WithField("market", marketStr).Debug("Token deposit belongs to desired market")
	return true
}
