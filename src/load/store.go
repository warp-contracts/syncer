package load

import (
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

// Periodically saves the states
// SinkTask handles caching data and periodically calling flush function
type Store struct {
	*task.SinkTask[*Payload]
	db *gorm.DB
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.SinkTask = task.NewSinkTask[*Payload](config, "store").
		WithOnFlush(500*time.Millisecond, self.save).
		WithBatchSize(15).
		WithBackoff(0, 10*time.Second)

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

func (self *Store) save(payloads []*Payload) error {
	self.Log.WithField("len", len(payloads)).Info("Saving payloads")

	err := self.db.Transaction(func(tx *gorm.DB) (err error) {
		for _, payload := range payloads {

			// self.Log.WithField("id", payload.Interaction.InteractionId).Info("Interaction")

			err = tx.Create(payload.Interaction).Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to save interaction")
				continue
			}

			payload.BundleItem.InteractionID = payload.Interaction.ID
			payload.BundleItem.State = model.BundleStatePending
			payload.BundleItem.BundlrResponse.Status = pgtype.Null

			err = tx.Create(payload.BundleItem).Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to save bundle item")
				continue
			}
		}
		return
	})

	if err != nil {
		self.Log.WithError(err).Error("Failed to save interaction")
		return nil
	}
	self.Log.WithField("len", len(payloads)).Info("Saved payloads")

	return nil
}
