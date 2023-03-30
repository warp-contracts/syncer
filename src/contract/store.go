package contract

import (
	"syncer/src/utils/config"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically saves the contracts and contract sources
// SinkTask handles caching data and periodically calling flush function
// This way we minimize the number of database transactions
type Store struct {
	*task.SinkTask[*Payload]
	db      *gorm.DB
	monitor monitoring.Monitor
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.SinkTask = task.NewSinkTask[*Payload](config, "store").
		WithOnFlush(config.Contract.StoreInterval, self.save).
		WithBatchSize(config.Contract.StoreBatchSize)
	return
}

func (self *Store) WithDB(db *gorm.DB) *Store {
	self.db = db
	return self
}

func (self *Store) WithInputChannel(input chan *Payload) *Store {
	self.SinkTask = self.SinkTask.WithInputChannel(input)
	return self
}

func (self *Store) WithMonitor(monitor monitoring.Monitor) *Store {
	self.monitor = monitor
	return self
}

func (self *Store) save(payloads []*Payload) error {
	self.Log.WithField("len", len(payloads)).Debug("Saving payloads")

	err := self.db.Transaction(func(tx *gorm.DB) (err error) {
		for _, payload := range payloads {
			err = tx.Create(payload.Contract).Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to save contract")
				continue
			}

			err = tx.Create(payload.Source).Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to save bundle item")
				continue
			}
		}
		return
	})

	if err != nil {
		self.Log.WithError(err).Error("Failed to save contracts")
	} else {
		self.Log.WithField("len", len(payloads)).Info("Saved payloads")
	}

	return nil
}
