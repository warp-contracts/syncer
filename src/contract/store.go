package contract

import (
	"errors"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"

	"time"

	"github.com/gammazero/deque"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles saving data to the database in na robust way.
// - groups incoming contracts into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Store struct {
	*task.Processor[*Payload, *ContractData]

	DB *gorm.DB

	monitor monitoring.Monitor

	contractFinishedHeight uint64
	lastProcessedBlockHash []byte
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Processor = task.NewProcessor[*Payload, *ContractData](config, "store-contract").
		WithBatchSize(config.Contract.StoreBatchSize).
		WithOnFlush(time.Second, self.flush).
		WithOnProcess(self.process).
		WithMaxBackoffInterval(15 * time.Second)

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
	self.contractFinishedHeight = payload.BlockHeight
	self.lastProcessedBlockHash = payload.BlockHash
	data = payload.Data
	return
}

func (self *Store) flush(data deque.Deque[*ContractData]) (err error) {
	return self.DB.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) error {
			if self.contractFinishedHeight <= 0 {
				return errors.New("block height too small")
			}

			state := model.State{
				Id:                        1,
				ContractFinishedHeight:    self.contractFinishedHeight,
				ContractFinishedBlockHash: self.lastProcessedBlockHash,
			}
			err = tx.WithContext(self.Ctx).
				Save(&state).
				Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to update state after last block")
				return err
			}

			if data.Len() <= 0 {
				// No contract to insert
				return nil
			}

			for i := 0; i < data.Len(); i++ {
				d := data.At(i)

				// Insert contract
				err = tx.WithContext(self.Ctx).
					Clauses(clause.OnConflict{DoNothing: true}).
					Create(d.Contract).
					Error
				if err != nil {
					//  FIXME: Monitor
					self.Log.WithError(err).Error("Failed to insert contract")
					continue
				}

				// Insert Source
				err = tx.WithContext(self.Ctx).
					Clauses(clause.OnConflict{DoNothing: true}).
					Create(d.Source).
					Error
				if err != nil {
					//  FIXME: Monitor
					self.Log.WithError(err).Error("Failed to insert contract source")
					continue
				}
				// FIXME: Monitor
			}

			return nil
		})
}
