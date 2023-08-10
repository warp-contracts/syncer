package sync

import (
	"errors"
	"time"

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
	*task.Processor[*Payload, *model.Interaction]

	DB      *gorm.DB
	monitor monitoring.Monitor

	savedBlockHeight  uint64
	finishedTimestamp uint64
	finishedHeight    uint64
	finishedBlockHash []byte

	replaceExistingData bool
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Processor = task.NewProcessor[*Payload, *model.Interaction](config, "store").
		WithBatchSize(config.Syncer.StoreBatchSize).
		WithOnFlush(config.Syncer.StoreMaxTimeInQueue, self.flush).
		WithOnProcess(self.process).
		WithBackoff(0, config.Syncer.StoreMaxBackoffInterval)

	return
}

func (self *Store) WithMonitor(v monitoring.Monitor) *Store {
	self.monitor = v
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

func (self *Store) WithReplaceExistingData(replace bool) *Store {
	self.replaceExistingData = replace
	return self
}

func (self *Store) process(payload *Payload) (out []*model.Interaction, err error) {
	self.finishedTimestamp = payload.BlockTimestamp
	self.finishedHeight = payload.BlockHeight
	self.finishedBlockHash = payload.BlockHash
	out = payload.Interactions
	return
}

func (self *Store) flush(data []*model.Interaction) (out []*model.Interaction, err error) {
	if self.savedBlockHeight == self.finishedHeight && len(data) == 0 {
		// No need to flush, nothing changed
		return
	}

	if self.finishedHeight <= 0 {
		err = errors.New("block height too small")
		return
	}

	self.Log.WithField("count", len(data)).Debug("Flushing interactions")
	defer self.Log.Trace("Flushing interactions done")

	// Set sync timestamp
	now := time.Now().UnixMilli()
	for _, interaction := range data {
		err = interaction.SyncTimestamp.Set(now)
		if err != nil {
			self.Log.WithError(err).Error("Failed to set sync_timestamp")
			return
		}
	}

	err = self.DB.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) error {
			err = self.updateFinishedBlock(tx)
			if err != nil {
				return err
			}

			return self.saveInteractions(tx, data)
		})
	if err != nil {
		return
	}

	// Successfuly saved interactions
	self.monitor.GetReport().Syncer.State.InteractionsSaved.Add(uint64(len(data)))

	// Update saved block height
	self.savedBlockHeight = self.finishedHeight

	self.monitor.GetReport().Syncer.State.FinishedHeight.Store(int64(self.savedBlockHeight))

	// Processing stops here, no need to return anything
	out = nil
	return
}

func (self *Store) updateFinishedBlock(tx *gorm.DB) (err error) {
	var state model.State
	err = tx.WithContext(self.Ctx).
		Where("name = ?", model.SyncedComponentInteractions).
		First(&state).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get state")
		self.monitor.GetReport().Contractor.Errors.DbLastTransactionBlockHeight.Inc()
		return err
	}

	// Replace finished block info, if it's newer
	if state.FinishedBlockHeight < self.finishedHeight {
		err = tx.WithContext(self.Ctx).
			Model(&model.State{
				Name: model.SyncedComponentInteractions,
			}).
			Updates(model.State{
				FinishedBlockTimestamp: self.finishedTimestamp,
				FinishedBlockHeight:    self.finishedHeight,
				FinishedBlockHash:      self.finishedBlockHash,
			}).
			Error
		if err != nil {
			self.Log.WithError(err).Error("Failed to update last transaction block height")
			self.monitor.GetReport().Syncer.Errors.DbLastTransactionBlockHeightError.Inc()
			return err
		}
	}

	return
}

func (self *Store) saveInteractions(tx *gorm.DB, data []*model.Interaction) (err error) {
	if len(data) == 0 {
		return nil
	}

	var onConflict clause.Expression
	if !self.replaceExistingData {
		// Do nothing upon conflict
		onConflict = clause.OnConflict{
			DoNothing: true,
			Columns:   []clause.Column{{Name: "interaction_id"}},
			UpdateAll: false,
		}
	} else {
		// We want to replace a subset of columns upon conflict
		onConflict = clause.OnConflict{
			DoNothing: false,
			Columns:   []clause.Column{{Name: "interaction_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"interaction",
				"block_height",
				"block_id",
				"contract_id",
				"function",
				"input",
				"confirmation_status",
				"confirming_peer",
				"confirmed_at_height",
				"confirmations",
				"source",
				// "bundler_tx_id",
				"interact_write",
				// "sort_key",
				"evolve",
				"testnet",
				// "last_sort_key",
				"owner",
				"block_timestamp",
			}), // column needed to be updated
		}
	}

	err = tx.WithContext(self.Ctx).
		Table("interactions").
		Clauses(onConflict).
		CreateInBatches(data, len(data)).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to insert Interactions")
		self.Log.WithField("interactions", data).Debug("Failed interactions")
		self.monitor.GetReport().Syncer.Errors.DbInteractionInsert.Inc()
		return err
	}
	return nil
}
