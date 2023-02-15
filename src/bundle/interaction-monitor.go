package bundle

import (
	"context"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"gorm.io/gorm"
)

// Gets the unbundled interactions and puts them on the output channel
type InteractionMonitor struct {
	*task.Task
	db *gorm.DB

	// Data about the interactions that need to be bundled
	BundleItems chan []model.BundleItem
}

func NewInteractionMonitor(config *config.Config, db *gorm.DB) (self *InteractionMonitor) {
	self = new(InteractionMonitor)
	self.db = db

	self.BundleItems = make(chan []model.BundleItem)

	self.Task = task.NewTask(config, "interaction-monitor").
		WithSubtaskFunc(self.run).
		// Worker pool for downloading interactions in parallel
		// Pool of workers that actually do the check.
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.InteractionManagerMaxParallelQueries).
		WithOnAfterStop(func() {
			close(self.BundleItems)
		})

	return
}

func (self *InteractionMonitor) run() error {
	var timer *time.Timer
	f := func() {
		// Setup waiting before the next check
		defer func() { timer = time.NewTimer(time.Second * 10) }()

		self.Log.Debug("Tick")
		if self.Workers.WaitingQueueSize() > 1 {
			self.Log.Debug("Too many pending interaction checks")
			return
		}

		self.Workers.Submit(self.check)
	}

	for {
		f()
		select {
		case <-self.StopChannel:
			self.Log.Debug("Stop monitoring interactions")
			return nil
		case <-timer.C:
			// pass through
		}
	}
}

func (self *InteractionMonitor) check() {
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

	select {
	case <-self.StopChannel:
	case self.BundleItems <- bundleItems:
	}
}
