package bundle

import (
	"context"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets the unbundled interactions and puts them on the output channel
// Gets interactions that somehow didn't get sent through the notification channel.
// Probably because of a restart.
type Poller struct {
	*task.Task
	db *gorm.DB

	// Data about the interactions that need to be bundled
	bundleItems chan *model.BundleItem
}

func NewPoller(config *config.Config) (self *Poller) {
	self = new(Poller)

	self.Task = task.NewTask(config, "poller").
		WithPeriodicSubtaskFunc(config.Bundler.PollerInterval, self.runPeriodically).
		// Worker pool for downloading interactions in parallel
		// Pool of workers that actually do the check.
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.InteractionManagerMaxParallelQueries)

	return
}

func (self *Poller) WithDB(db *gorm.DB) *Poller {
	self.db = db
	return self
}

func (self *Poller) WithOutputChannel(bundleItems chan *model.BundleItem) *Poller {
	self.bundleItems = bundleItems
	return self
}

func (self *Poller) runPeriodically() error {
	self.Log.Debug("Tick")
	if self.Workers.WaitingQueueSize() > 1 {
		self.Log.Debug("Too many pending interaction checks")
		return nil
	}

	self.Workers.Submit(self.check)

	return nil
}

func (self *Poller) check() {
	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.Bundler.PollerTimeout)
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
		RETURNING *`, self.Config.Bundler.ConfirmerMaxBatchSize).
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
