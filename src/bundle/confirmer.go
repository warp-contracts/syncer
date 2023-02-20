package bundle

import (
	"sync"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"github.com/gammazero/deque"
	"gorm.io/gorm"
)

// Periodically saves the confirmation state of the bundlet interactions
// This is done to prevent flooding database with bundle_items state updates
type Confirmer struct {
	*task.Task
	db *gorm.DB

	// Data about the interactions that need to be bundled
	bundled chan *Confirmation

	// Ids of the bundle items that will be confirmed
	confirmations deque.Deque[*Confirmation]

	mtx sync.RWMutex
}

type Confirmation struct {
	InteractionID int
	BundlerTxID   string
	Signature     string
}

func NewConfirmer(config *config.Config) (self *Confirmer) {
	self = new(Confirmer)

	self.confirmations.SetMinCapacity(uint(config.Bundler.ConfirmerMaxBatchSize) * 2)

	self.Task = task.NewTask(config, "confirmer").
		WithSubtaskFunc(self.receive).
		WithPeriodicSubtaskFunc(time.Second, self.updatePeriodically)

	return
}

func (self *Confirmer) WithDB(db *gorm.DB) *Confirmer {
	self.db = db
	return self
}

func (self *Confirmer) WithInputChannel(bundled chan *Confirmation) *Confirmer {
	self.bundled = bundled
	return self
}

func (self *Confirmer) numPendingConfirmation() int {
	self.mtx.RLock()
	defer self.mtx.RUnlock()
	return self.confirmations.Len()
}

func (self *Confirmer) receive() error {
	for id := range self.bundled {
		self.mtx.Lock()
		self.confirmations.PushBack(id)
		self.mtx.Unlock()
	}
	return nil
}

func (self *Confirmer) updatePeriodically() error {
	if self.numPendingConfirmation() > 10000 {
		self.Log.WithField("len", self.confirmations.Len()).Warn("Too many interactions to confirm")
	}

	// Repeat while there are still confirmations in the queue
	for {
		self.mtx.Lock()
		size := self.confirmations.Len()
		if size == 0 {
			self.mtx.Unlock()
			break
		}

		if size > self.Config.Bundler.ConfirmerMaxBatchSize {
			size = self.Config.Bundler.ConfirmerMaxBatchSize
		}

		// Copy data to avoid locking for too long
		confirmations := make([]*Confirmation, 0, size)
		// Fill batch
		for i := 0; i < size; i++ {
			confirmations = append(confirmations, self.confirmations.PopFront())
		}
		self.mtx.Unlock()

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
			return err
		}
	}
	return nil
}
