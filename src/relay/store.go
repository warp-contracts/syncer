package relay

import (
	"errors"
	"time"

	"github.com/cometbft/cometbft/libs/bytes"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles saving data to the database in na robust way.
// - groups incoming Interactions into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Store struct {
	*task.Processor[*Payload, *Payload]

	DB      *gorm.DB
	monitor monitoring.Monitor

	savedBlockHeight  uint64
	finishedTimestamp uint64
	finishedHeight    uint64
	finishedBlockHash bytes.HexBytes

	isReplacing bool
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Processor = task.NewProcessor[*Payload, *Payload](config, "store").
		WithBatchSize(config.Relayer.StoreBatchSize).
		WithOnFlush(config.Relayer.StoreMaxTimeInQueue, self.flush).
		WithOnProcess(self.process).
		WithBackoff(0, config.Relayer.StoreMaxBackoffInterval)

	return
}

func (self *Store) WithMonitor(v monitoring.Monitor) *Store {
	self.monitor = v
	return self
}

func (self *Store) WithIsReplacing(v bool) *Store {
	self.isReplacing = v
	return self
}

func (self *Store) WithInputChannel(v chan *Payload) *Store {
	self.Processor = self.Processor.WithInputChannel(v)
	return self
}

func (self *Store) WithDB(v *gorm.DB) *Store {
	self.DB = v
	return self
}

func (self *Store) process(payload *Payload) (out []*Payload, err error) {
	// self.Log.WithField("height", payload.SequencerBlockHeight).Debug("Processing")
	self.finishedTimestamp = uint64(payload.SequencerBlockTimestamp)
	self.finishedHeight = uint64(payload.SequencerBlockHeight)
	self.finishedBlockHash = payload.SequencerBlockHash
	out = []*Payload{payload}
	return
}

func (self *Store) flush(payloads []*Payload) (out []*Payload, err error) {
	if self.savedBlockHeight == self.finishedHeight && len(payloads) == 0 {
		// No need to flush, nothing changed
		return
	}

	self.Log.WithField("height", self.finishedHeight).WithField("num", len(payloads)).Debug("-> Saving blocks")
	defer self.Log.WithField("height", self.finishedHeight).WithField("num", len(payloads)).Debug("<- Saving blocks")

	if self.finishedHeight <= 0 {
		err = errors.New("block height too small")
		return
	}

	// Interactions from all blocks
	var (
		lastArweaveBlock    *ArweaveBlock
		arweaveInteractions []*model.Interaction
		interactions        []*model.Interaction
		bundleItems         []*model.BundleItem
	)

	// Concatenate data from all payloads
	for _, payload := range payloads {
		if len(payload.ArweaveBlocks) > 0 {
			for _, arweaveBlock := range payload.ArweaveBlocks {
				arweaveInteractions = append(arweaveInteractions, arweaveBlock.Interactions...)
			}
			lastArweaveBlock = payload.ArweaveBlocks[len(payload.ArweaveBlocks)-1]
		}
		interactions = append(interactions, payload.Interactions...)
		bundleItems = append(bundleItems, payload.BundleItems...)
	}

	// Set sync timestamp
	now := time.Now().UnixMilli()
	for _, interaction := range interactions {
		err = interaction.SyncTimestamp.Set(now)
		if err != nil {
			self.Log.WithError(err).Error("Failed to set sync_timestamp")
			return
		}
	}
	for _, interaction := range arweaveInteractions {
		err = interaction.SyncTimestamp.Set(now)
		if err != nil {
			self.Log.WithError(err).Error("Failed to set sync_timestamp")
			return
		}
	}

	if len(interactions) != len(bundleItems) {
		err = errors.New("bundle items and interactions count mismatch")
		self.Log.WithError(err).Error("Bundle items and interactions count mismatch")
		return
	}

	err = self.DB.WithContext(self.Ctx).
		// Debug().
		Transaction(func(tx *gorm.DB) error {
			err = tx.WithContext(self.Ctx).
				Model(&model.State{
					Name: model.SyncedComponentRelayer,
				}).
				// State struct accepts block hash as bytes, but here we have only a string
				Updates(map[string]interface{}{
					"finished_block_timestamp": self.finishedTimestamp,
					"finished_block_height":    self.finishedHeight,
					"finished_block_hash":      self.finishedBlockHash.String(),
				}).
				Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to update last transaction block height")
				return err
			}

			if lastArweaveBlock != nil {
				// Save L1 interactions if there are any
				err = tx.WithContext(self.Ctx).
					Table(model.TableInteraction).
					Clauses(clause.OnConflict{
						DoNothing: !self.isReplacing,
						Columns:   []clause.Column{{Name: "interaction_id"}},
						UpdateAll: self.isReplacing,
					}, clause.Returning{Columns: []clause.Column{{Name: "id"}}}).
					CreateInBatches(arweaveInteractions, self.Config.Relayer.StoreBatchSize).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to insert Interactions")
					// self.Log.WithField("interactions", interactions).Debug("Failed interactions")
					return err
				}

				// Arweave interactions are saved in this transactions
				err = self.updateFinishedArweaveBlock(tx, lastArweaveBlock)
				if err != nil {
					return err
				}
			}

			if len(interactions) != 0 {
				// Save L2 interactions if there are any
				err = tx.WithContext(self.Ctx).
					Clauses(clause.OnConflict{
						DoNothing: !self.isReplacing,
						Columns:   []clause.Column{{Name: "interaction_id"}},
						UpdateAll: self.isReplacing,
					}).
					CreateInBatches(&interactions, self.Config.Relayer.StoreBatchSize).
					Error

				if err != nil {
					self.Log.WithError(err).Error("Failed to insert Interactions")
					return err
				}

				// Connect bundle items with interactions
				// L1 interactions don't have corresponding bundle items
				// TODO: Handle bundle items with arweave interactions order
				for idx := range interactions {
					bundleItems[idx].InteractionID = interactions[idx].ID
				}
			}

			if len(bundleItems) != 0 {
				// Save bundle items if there are any
				err = tx.WithContext(self.Ctx).
					Table(model.TableBundleItem).
					Clauses(clause.OnConflict{
						DoNothing: true,
						Columns:   []clause.Column{{Name: "interaction_id"}},
						UpdateAll: false,
					}).
					CreateInBatches(bundleItems, self.Config.Relayer.StoreBatchSize).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to insert bundle items")
					return err
				}
			}

			return nil
		})
	if err != nil {
		self.Log.WithError(err).Error("Failed to insert bundle items, interactions and update state")
		return
	}

	// Update saved block height
	self.savedBlockHeight = self.finishedHeight

	// Successfuly saved interactions
	self.monitor.GetReport().Relayer.State.InteractionsSaved.Add(uint64(len(interactions) + len(arweaveInteractions)))
	self.monitor.GetReport().Relayer.State.L1InteractionsSaved.Add(uint64(len(arweaveInteractions)))
	self.monitor.GetReport().Relayer.State.L2InteractionsSaved.Add(uint64(len(interactions)))

	// Bundle items saved
	self.monitor.GetReport().Relayer.State.BundleItemsSaved.Store(uint64(len(bundleItems)))

	// Finished height of sequencer and arweave
	self.monitor.GetReport().Relayer.State.SequencerFinishedHeight.Store(int64(self.savedBlockHeight))
	if lastArweaveBlock != nil {
		self.monitor.GetReport().Relayer.State.ArwaeveFinishedHeight.Store(int64(lastArweaveBlock.Block.Height))
	}

	// Processing stops here, no need to return anything
	out = nil

	return
}

func (self *Store) updateFinishedArweaveBlock(tx *gorm.DB, arweaveBlock *ArweaveBlock) (err error) {
	var state model.State
	err = tx.WithContext(self.Ctx).
		Where("name = ?", model.SyncedComponentInteractions).
		First(&state).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get state")
		self.monitor.GetReport().Relayer.Errors.DbError.Inc()
		return err
	}

	// Replace finished block info, if it's newer
	if state.FinishedBlockHeight < uint64(arweaveBlock.Block.Height) {
		err = tx.WithContext(self.Ctx).
			Model(&model.State{
				Name: model.SyncedComponentInteractions,
			}).
			Updates(model.State{
				FinishedBlockTimestamp: uint64(arweaveBlock.Block.Timestamp),
				FinishedBlockHeight:    uint64(arweaveBlock.Block.Height),
				FinishedBlockHash:      arweaveBlock.Block.Hash,
			}).
			Error
		if err != nil {
			self.Log.WithError(err).Error("Failed to update last transaction block height")
			self.monitor.GetReport().Relayer.Errors.DbError.Inc()
			return err
		}
	}

	return
}
