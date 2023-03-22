package bundle

import (
	"context"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets the unbundled interactions and puts them on the output channel
// Gets interactions that somehow didn't get sent through the notification channel.
// Probably because of a restart.
type Poller struct {
	*task.Task
	db *gorm.DB

	monitor monitoring.Monitor

	// Data about the interactions that need to be bundled
	output chan *model.BundleItem
}

func NewPoller(config *config.Config) (self *Poller) {
	self = new(Poller)

	self.Task = task.NewTask(config, "poller").
		WithPeriodicSubtaskFunc(config.Bundler.PollerInterval, self.runPeriodically).
		// Worker pool for downloading interactions in parallel
		// Pool of workers that actually do the check.
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.InteractionManagerMaxParallelQueries, 1)

	return
}

func (self *Poller) WithDB(db *gorm.DB) *Poller {
	self.db = db
	return self
}

func (self *Poller) WithOutputChannel(bundleItems chan *model.BundleItem) *Poller {
	self.output = bundleItems
	return self
}

func (self *Poller) WithMonitor(monitor monitoring.Monitor) *Poller {
	self.monitor = monitor
	return self
}

func (self *Poller) runPeriodically() error {
	self.SubmitToWorker(self.check)
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
			OR (state = 'UPDATING'::bundle_state AND EXTRACT(EPOCH FROM (NOW() - updated_at)) < ?)
			ORDER BY interaction_id ASC
			LIMIT ?
		)
		UPDATE bundle_items
		SET state = 'UPLOADING'::bundle_state, updated_at = NOW()
		WHERE interaction_id IN (SELECT interaction_id FROM rows)
		RETURNING *`, self.Config.Bundler.ConfirmerRetryBundleAfter.Seconds(), self.Config.Bundler.ConfirmerMaxBatchSize).
		Scan(&bundleItems).Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get interactions")
	}

	self.Log.WithField("count", len(bundleItems)).Debug("Polled bundle items")

	for i := range bundleItems {
		select {
		case <-self.StopChannel:
			return
		case self.output <- &bundleItems[i]:
		}
	}

	// Update metrics
	self.monitor.GetReport().Bundler.State.BundlesFromSelects.Add(uint64(len(bundleItems)))
}
