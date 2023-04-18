package contract

import (
	"errors"
	"sync"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles saving data to the database in na robust way.
// - groups incoming contracts into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Store struct {
	*task.Processor[*Payload, *ContractData]

	DB  *gorm.DB
	mtx sync.Mutex

	monitor monitoring.Monitor

	savedBlockHeight          uint64
	contractFinishedHeight    uint64
	contractFinishedBlockHash []byte
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Processor = task.NewProcessor[*Payload, *ContractData](config, "store-contract").
		WithBatchSize(config.Contract.StoreBatchSize).
		WithOnFlush(config.Contract.StoreInterval, self.flush).
		WithOnProcess(self.process).
		WithBackoff(config.Contract.StoreBackoffMaxElapsedTime, config.Contract.StoreBackoffMaxInterval)

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

func (self *Store) process(payload *Payload) (data []*ContractData, err error) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	self.contractFinishedHeight = payload.BlockHeight
	self.contractFinishedBlockHash = payload.BlockHash
	data = payload.Data
	return
}

func (self *Store) getState(payload *Payload) (savedBlockHeight, contractFinishedHeight uint64, contractFinishedBlockHash []byte) {
	self.mtx.Lock()
	defer self.mtx.Unlock()
	return self.savedBlockHeight, self.contractFinishedHeight, self.contractFinishedBlockHash
}

func (self *Store) flush(data []*ContractData) (out []*ContractData, err error) {
	savedBlockHeight, contractFinishedHeight, contractFinishedBlockHash := self.getState(nil)

	if savedBlockHeight == contractFinishedHeight && len(data) == 0 {
		// No need to flush, nothing changed
		return
	}

	self.Log.WithField("count", len(data)).Trace("Flushing contracts")
	defer self.Log.Trace("Flushing contracts done")

	err = self.DB.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) error {
			if contractFinishedHeight <= 0 {
				return errors.New("block height too small")
			}

			err = tx.WithContext(self.Ctx).
				Model(&model.State{
					Name: model.SyncedComponentContracts,
				}).
				Updates(model.State{
					FinishedBlockHeight: contractFinishedHeight,
					FinishedBlockHash:   contractFinishedBlockHash,
				}).
				Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to update state after last block")
				self.monitor.GetReport().Contractor.Errors.DbLastTransactionBlockHeight.Inc()
				return err
			}

			for _, d := range data {
				// Insert contract
				err = tx.WithContext(self.Ctx).
					Clauses(clause.OnConflict{DoNothing: true}).
					Create(d.Contract).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to insert contract")
					self.monitor.GetReport().Contractor.Errors.DbContractInsert.Inc()
					continue
				}

				// Insert Source
				err = tx.WithContext(self.Ctx).
					Clauses(clause.OnConflict{DoNothing: true}).
					Create(d.Source).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to insert contract source")
					self.monitor.GetReport().Contractor.Errors.DbSourceInsert.Inc()
					continue
				}

			}

			return nil
		})
	if err != nil {
		return
	}

	self.monitor.GetReport().Contractor.State.ContractsSaved.Add(uint64(len(data)))

	// Update saved block height
	self.mtx.Lock()
	self.savedBlockHeight = self.contractFinishedHeight
	self.mtx.Unlock()

	self.monitor.GetReport().Contractor.State.FinishedHeight.Store(self.savedBlockHeight)

	// Everything went well, pass all data further
	out = data
	return
}
