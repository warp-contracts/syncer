package redstone_tx_sync

import (
	"database/sql"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Store is responsible for saving last synced block height in sync_state table
// It updates the table periodically to not overload database with updates done each time after new block has been processed
type Store struct {
	*task.Processor[*LastSyncedBlockPayload, *LastSyncedBlockPayload]

	db      *gorm.DB
	monitor monitoring.Monitor

	savedLastSyncedBlockHeight int64
	lastSyncedBlockHeight      int64
	lastSyncedBlockHash        string
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Processor = task.NewProcessor[*LastSyncedBlockPayload, *LastSyncedBlockPayload](config, "store").
		WithBatchSize(config.RedstoneTxSyncer.StoreBatchSize).
		WithOnFlush(config.RedstoneTxSyncer.StoreInterval, self.flush).
		WithOnProcess(self.process).
		WithBackoff(0, config.RedstoneTxSyncer.StoreMaxBackoffInterval)

	return
}

func (self *Store) WithMonitor(v monitoring.Monitor) *Store {
	self.monitor = v
	return self
}

func (self *Store) WithInputChannel(v chan *LastSyncedBlockPayload) *Store {
	self.Processor = self.Processor.WithInputChannel(v)
	return self
}

func (self *Store) WithDb(v *gorm.DB) *Store {
	self.db = v
	return self
}

func (self *Store) process(payload *LastSyncedBlockPayload) (out []*LastSyncedBlockPayload, err error) {
	self.lastSyncedBlockHeight = payload.Height
	self.lastSyncedBlockHash = payload.Hash
	return
}

func (self *Store) flush([]*LastSyncedBlockPayload) (out []*LastSyncedBlockPayload, err error) {
	if self.savedLastSyncedBlockHeight == self.lastSyncedBlockHeight {
		// No need to flush, nothing changed
		return
	}

	self.Log.WithField("height", self.lastSyncedBlockHeight).Debug("Updating sync_state with last synced block height")
	defer self.Log.Trace("Updating sync_state with last synced block height done")

	err = self.db.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) error {
			err = self.updateLastSyncedHeight(tx)
			return err
		}, &sql.TxOptions{Isolation: sql.LevelRepeatableRead})
	if err != nil {
		return
	}

	// Update saved block height
	self.savedLastSyncedBlockHeight = self.lastSyncedBlockHeight

	self.monitor.GetReport().RedstoneTxSyncer.State.StoreLastSyncedBlockHeight.Store(int64(self.savedLastSyncedBlockHeight))

	// Processing stops here, no need to return anything
	out = nil
	return
}

func (self *Store) updateLastSyncedHeight(tx *gorm.DB) (err error) {
	var state model.State
	err = tx.WithContext(self.Ctx).
		Where("name = ?", model.SyncedComponentInteractions).
		First(&state).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get state")
		self.monitor.GetReport().RedstoneTxSyncer.Errors.StoreGetLastStateFailure.Inc()
		return
	}

	// Replace finished block info, if it's newer
	if state.FinishedBlockHeight < uint64(self.lastSyncedBlockHeight) {
		err = tx.WithContext(self.Ctx).
			Model(&model.State{
				Name: model.SyncedComponentRedstoneTxSyncer,
			}).
			Updates(model.State{
				FinishedBlockHeight: uint64(self.lastSyncedBlockHeight),
				FinishedBlockHash:   arweave.Base64String(self.lastSyncedBlockHash),
			}).
			Error
		if err != nil {
			self.Log.WithError(err).Error("Failed to update last synced block height")
			self.monitor.GetReport().RedstoneTxSyncer.Errors.StoreSaveLastStateFailure.Inc()
			return
		}
	}

	return
}
