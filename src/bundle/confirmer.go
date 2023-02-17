package bundle

import (
	"context"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"gorm.io/gorm"
)

// Periodically saves the confirmation state of the bundlet interactions
// This is done to prevent flooding database with bundle_items state updates
type Confirmer struct {
	*task.Task
	db *gorm.DB

	// Data about the interactions that need to be bundled
	bundled chan *model.BundleItem
}

func NewConfirmer(config *config.Config) (self *Confirmer) {
	self = new(Confirmer)

	self.Task = task.NewTask(config, "confirmer").
		WithPeriodicSubtaskFunc(time.Second, self.runPeriodically)

	return
}

func (self *Confirmer) WithDB(db *gorm.DB) *Confirmer {
	self.db = db
	return self
}

func (self *Confirmer) WithInputChannel(bundled chan *model.BundleItem) *Confirmer {
	self.bundled = bundled
	return self
}

func (self *Confirmer) runPeriodically() error {
	self.Log.Debug("Tick")
	if self.Workers.WaitingQueueSize() > 1 {
		self.Log.Debug("Too many pending interaction checks")
		return nil
	}

	self.Workers.Submit(self.check)

	return nil
}

func (self *Confirmer) check() {
	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.InteractionManagerTimeout)
	defer cancel()

	// Inserts interactions that weren't yet bundled into bundle_items table
	var bundleItems []model.BundleItem
	err := self.db.WithContext(ctx).
		Raw(`WITH rows AS (
			SELECT interaction_id
			FROM bundle_items
			WHERE state = 'PENDING'::bundle_state
			ORDER BY interaction_id ASC
			LIMIT ?
		)
		UPDATE bundle_items
		SET state = 'UPLOADING'::bundle_state
		WHERE interaction_id IN (SELECT interaction_id FROM rows)
		RETURNING *`, 10).
		Scan(&bundleItems).Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get interactions")
	}

	for i := range bundleItems {
		select {
		case <-self.StopChannel:
		case self.bundleItems <- &bundleItems[i]:
		}
	}
}
