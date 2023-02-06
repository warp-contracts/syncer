package bundle

import (
	"context"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"github.com/gammazero/workerpool"
	"gorm.io/gorm"
)

// Gets the unbundled interactions and puts them on the output channel
type InteractionMonitor struct {
	*task.Task
	db *gorm.DB

	// Pool of workers that actually do the check.
	// It's possible to run multiple requests in parallel.
	// We're limiting the number of parallel requests with the number of workers.
	workers *workerpool.WorkerPool

	// Data about the interactions that need to be bundled
	BundleItems chan []model.BundleItem
}

func NewInteractionMonitor(config *config.Config, db *gorm.DB) (self *InteractionMonitor) {
	self = new(InteractionMonitor)
	self.db = db

	self.BundleItems = make(chan []model.BundleItem)

	self.Task = task.NewTask(config, "interaction-monitor").
		WithSubtaskFunc(self.run).
		WithOnStop(func() {
			self.workers.Stop()
		}).
		WithOnAfterStop(func() {
			close(self.BundleItems)
		})

	// Worker pool for downloading interactions in parallel
	self.workers = workerpool.New(self.Config.InteractionManagerMaxParallelQueries)

	return
}

func (self *InteractionMonitor) run() error {
	var timer *time.Timer
	f := func() {
		// Setup waiting before the next check
		defer func() { timer = time.NewTimer(time.Second * 10) }()

		self.Log.Debug("Tick")
		if self.workers.WaitingQueueSize() > 1 {
			self.Log.Debug("Too many pending interaction checks")
			return
		}

		self.workers.Submit(self.check)
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
		Raw(`INSERT INTO bundle_items
	SELECT interactions.id as interaction_id, 'UPLOADING' as state, NULL as block_height, CURRENT_TIMESTAMP as updated_at
	FROM interactions
	LEFT JOIN bundle_items ON interactions.id = bundle_items.interaction_id
	WHERE bundle_items.interaction_id IS NULL 
	AND interactions.source='redstone-sequencer'
	AND interactions.id > 400 
	ORDER BY interactions.id
	LIMIT ?
	RETURNING *	`, 10).
		Scan(&bundleItems).Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get interactions")
	}

	select {
	case <-self.StopChannel:
	case self.BundleItems <- bundleItems:
	}
}
