package bundle

import (
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"gorm.io/gorm"
)

// Periodically saves the confirmation state of the bundlet interactions
// This is done to prevent flooding database with bundle_items state updates
type Confirmer struct {
	*task.SinkTask[*Confirmation]
	db *gorm.DB
}

type Confirmation struct {
	InteractionID int
	BundlerTxID   string
	Signature     string
}

func NewConfirmer(config *config.Config) (self *Confirmer) {
	self = new(Confirmer)

	self.SinkTask = task.NewSinkTask[*Confirmation](config, "confirmer").
		WithBatchSize(100).
		WithOnFlush(time.Second, self.save)

	return
}

func (self *Confirmer) WithDB(db *gorm.DB) *Confirmer {
	self.db = db
	return self
}

func (self *Confirmer) WithInputChannel(input chan *Confirmation) *Confirmer {
	self.SinkTask.WithInputChannel(input)
	return self
}

func (self *Confirmer) save(confirmations []*Confirmation) error {
	// Uses one transaction to do all the updates
	// NOTE: It still uses many requests to the database,
	// it should be possible to combine updates into batches, but it's not a priority for now.
	err := self.db.Transaction(func(tx *gorm.DB) error {
		for _, confirmation := range confirmations {
			err := tx.Model(&model.BundleItem{}).
				Where("interaction_id = ?", confirmation.InteractionID).
				Where("state = ?", model.BundleStateUploading).
				Update("state", model.BundleStateUploaded).
				Error
			if err != nil {
				return err
			}

			err = tx.Model(&model.Interaction{}).
				Where("id = ?", confirmation.InteractionID).
				Update("bundler_tx_id", confirmation.BundlerTxID).
				Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		self.Log.WithError(err).Error("Failed to save bundle items")
	}

	return err
}
