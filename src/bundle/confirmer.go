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
	bundled chan int

	// Ids of the bundle items that will be confirmed
	bundledIds deque.Deque[int]

	mtx sync.RWMutex
}

func NewConfirmer(config *config.Config) (self *Confirmer) {
	self = new(Confirmer)

	self.Task = task.NewTask(config, "confirmer").
		WithSubtaskFunc(self.receive).
		WithPeriodicSubtaskFunc(time.Second, self.updatePeriodically)

	return
}

func (self *Confirmer) WithDB(db *gorm.DB) *Confirmer {
	self.db = db
	return self
}

func (self *Confirmer) WithInputChannel(bundled chan int) *Confirmer {
	self.bundled = bundled
	return self
}

func (self *Confirmer) numPendingConfirmation() int {
	self.mtx.RLock()
	defer self.mtx.RUnlock()
	return self.bundledIds.Len()
}

func (self *Confirmer) receive() error {
	for id := range self.bundled {
		self.mtx.Lock()
		self.bundledIds.PushBack(id)
		self.mtx.Unlock()
	}
	return nil
}

func (self *Confirmer) updatePeriodically() error {
	if self.numPendingConfirmation() > 10000 {
		self.Log.WithField("len", self.bundledIds.Len()).Warn("Too many interactions to confirm")
	}

	for {
		self.mtx.Lock()
		size := self.bundledIds.Len()
		if size == 0 {
			self.mtx.Unlock()
			break
		}

		if size > self.Config.Bundler.ConfirmerMaxBatchSize {
			size = self.Config.Bundler.ConfirmerMaxBatchSize
		}

		ids := make([]int, 0, size)

		// Fill batch
		for i := 0; i < size; i++ {
			ids = append(ids, self.bundledIds.PopFront())
		}
		self.mtx.Unlock()

		// Mark successfuly sent bundles as UPLOADED
		err := self.db.Model(&model.BundleItem{}).
			Where("interaction_id IN ?", ids).
			Where("state = ?", model.BundleStateUploading).
			Update("state", model.BundleStateUploaded).
			Error
		if err != nil {
			// TODO: Is there anything else we can do?
			self.Log.WithError(err).Error("Failed to mark bundle item as uploaded to Bundlr")
		}
	}
	return nil
}
