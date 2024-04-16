package warpy_sync

import (
	"context"
	"errors"
	"math"
	"math/big"
	"slices"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/jackc/pgtype"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

type StoreSommelier struct {
	*task.Task

	db      *gorm.DB
	monitor monitoring.Monitor
	input   chan *SommelierTransactionPayload
}

func NewStoreSommelier(config *config.Config) (self *StoreSommelier) {
	self = new(StoreSommelier)

	self.Task = task.NewTask(config, "store_sommelier").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.WarpySyncer.SyncerSommelierNumWorkers, config.WarpySyncer.SyncerSommelierWorkerQueueSize)

	return
}

func (self *StoreSommelier) WithInputChannel(v chan *SommelierTransactionPayload) *StoreSommelier {
	self.input = v
	return self
}

func (self *StoreSommelier) WithDB(db *gorm.DB) *StoreSommelier {
	self.db = db
	return self
}

func (self *StoreSommelier) WithMonitor(monitor monitoring.Monitor) *StoreSommelier {
	self.monitor = monitor
	return self
}

func (self *StoreSommelier) run() (err error) {
	for payload := range self.input {
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

				self.monitor.GetReport().WarpySyncer.Errors.StoreSommelierFailures.Inc()
				self.Log.WithError(err).WithField("txId", payload.Transaction.Hash().String()).
					Warn("Could not process transaction, retrying...")
				return err
			}).
			Run(func() error {
				err = self.db.WithContext(self.Ctx).
					Transaction(func(dbTx *gorm.DB) error {
						err = self.insertLog(dbTx, payload.Transaction, payload.FromAddress, payload.Block, payload.Method, payload.ParsedInput)
						if err != nil {
							self.Log.WithError(err).Error("Log insert has not been successful")
							return nil
						}

						var ethTxAssetsFieldName string
						if slices.Contains(self.Config.WarpySyncer.StoreSommelierWithdrawFunctions, payload.Method.Name) {
							ethTxAssetsFieldName = "shares"
						} else {
							ethTxAssetsFieldName = "assets"
						}
						err = self.insertAssets(dbTx, payload.Transaction, payload.FromAddress, eth.WeiToEther(payload.Input[ethTxAssetsFieldName].(*big.Int)), payload.Method.Name, payload.Block)
						if err != nil {
							return err
						}

						return nil
					})

				return err
			})

		//Update monitoring
		self.monitor.GetReport().WarpySyncer.State.StoreSommelierRecordsSaved.Inc()
		self.Log.WithField("tx_id", payload.Transaction.Hash().String()).WithField("from", payload.FromAddress).Info("New log and assets inserted")

	}
	return nil
}

func (self *StoreSommelier) insertLog(dbTx *gorm.DB, tx *types.Transaction, from string, block *BlockInfoPayload, method *abi.Method, input []byte) (err error) {
	chain, _ := self.Config.WarpySyncer.SyncerChain.RpcProviderUrl()
	transactionPayload := &model.WarpySyncerTransaction{
		TxId:           tx.Hash().String(),
		FromAddress:    from,
		ToAddress:      tx.To().String(),
		BlockHeight:    uint64(block.Height),
		BlockTimestamp: block.Timestamp,
		SyncTimestamp:  uint64(time.Now().Unix()),
		MethodName:     method.Name,
		Chain:          chain,
		Input:          pgtype.JSONB{Bytes: input, Status: pgtype.Present},
	}
	err = dbTx.WithContext(self.Ctx).
		Table(model.TableWarpySyncerTransactions).
		Create(transactionPayload).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Could not insert Warpy Syncer transaction")
		return
	}

	self.Log.WithField("from_address", from).
		WithField("tx_id", tx.Hash().String()).
		Debug("New log inserted")

	return
}

func (self *StoreSommelier) insertAssets(dbTx *gorm.DB, tx *types.Transaction, from string, assets float64, methodName string, block *BlockInfoPayload) (err error) {
	var transactionPayload *model.WarpySyncerAssets
	if slices.Contains(self.Config.WarpySyncer.StoreSommelierWithdrawFunctions, methodName) {
		assets = math.Ceil(assets*1000) / 1000
		self.Log.WithField("tx_id", tx.Hash().String()).WithField("assets_to_subtract", assets).Info("Redeem transaction require subtraction")
		var lastTxs []*struct {
			TxId   string
			Assets float64
		}
		err = dbTx.WithContext(self.Ctx).
			Table(model.TableWarpySyncerAssets).
			Select("tx_id, assets").
			Where("from_address = ?", from).
			Order("timestamp DESC").
			Scan(&lastTxs).
			Error

		if err != nil {
			return err
		}

		self.Log.WithField("txs", len(lastTxs)).Debug("Found transactions to redeem subtraction")

		if len(lastTxs) == 0 {
			self.Log.Debug("No transactions to redeem subtraction have been found")
			return
		} else {
			assetsToSubtract := assets
			currentIdx := 0
			for assetsToSubtract > 0 {
				if len(lastTxs) < currentIdx+1 {
					self.Log.Debug("No more transactions to subtract")
					return
				} else if lastTxs[currentIdx].Assets > assetsToSubtract {
					err = dbTx.WithContext(self.Ctx).
						Table(model.TableWarpySyncerAssets).
						Where("tx_id = ?", lastTxs[currentIdx].TxId).
						Update("assets", math.Round((lastTxs[currentIdx].Assets-assetsToSubtract)*1000)/1000).
						Error

					if err != nil {
						self.Log.WithError(err).WithField("tx_id", lastTxs[currentIdx].TxId).Error("Could not update last tx assets")
						return err
					}

					self.Log.WithField("assets_subtracted", assetsToSubtract).
						WithField("tx_id", lastTxs[currentIdx].TxId).
						WithField("initial_tx_assets", lastTxs[currentIdx].Assets).
						Debug("No more assets to subtract")
					return
				} else {
					var txId = &model.WarpySyncerAssets{TxId: lastTxs[currentIdx].TxId}

					err = dbTx.WithContext(self.Ctx).
						Table(model.TableWarpySyncerAssets).
						Delete(&txId).
						Error

					if err != nil {
						self.Log.WithError(err).WithField("tx_id", lastTxs[currentIdx].TxId).Error("Could not delete last tx")
						return err
					}

					self.Log.WithField("assets_subtracted", assetsToSubtract-lastTxs[currentIdx].Assets).
						WithField("tx_id", lastTxs[currentIdx].TxId).
						WithField("initial_tx_assets", lastTxs[currentIdx].Assets).
						Debug("Last tx deleted")

					assetsToSubtract -= lastTxs[currentIdx].Assets
					currentIdx++
				}
			}
		}
	} else {
		assets = math.Round(assets*1000) / 1000
		transactionPayload = &model.WarpySyncerAssets{
			TxId:        tx.Hash().String(),
			FromAddress: from,
			Assets:      assets,
			Timestamp:   block.Timestamp,
			Protocol:    eth.Sommelier.String(),
		}

		err = dbTx.WithContext(self.Ctx).
			Table(model.TableWarpySyncerAssets).
			Create(transactionPayload).
			Error

		if err != nil {
			self.Log.WithError(err).WithField("tx_id", tx.Hash().String()).Error("Could not insert new Warpy Syncer assets")
			return
		}

		self.Log.WithField("assets", assets).
			WithField("tx_id", tx.Hash().String()).
			Debug("New assets inserted")

	}

	return
}
