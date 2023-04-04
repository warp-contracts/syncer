package bundle

import (
	"context"
	"fmt"
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

	if config.Bundler.PollerDisabled {
		self.Task = task.NewTask(config, "poller")
		return
	}

	self.Task = task.NewTask(config, "poller").
		WithPeriodicSubtaskFunc(config.Bundler.PollerInterval, self.handleNewTransactions).
		WithPeriodicSubtaskFunc(config.Bundler.PollerInterval, self.handleRetrying)

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

func (self *Poller) handleNewTransactions() error {
	// Repeat until there are no more interactions to fetch
	for {
		ctx, cancel := context.WithTimeout(self.Ctx, self.Config.Bundler.PollerTimeout)
		defer cancel()

		// Inserts interactions that weren't yet bundled into bundle_items table
		var bundleItems []model.BundleItem
		err := self.db.WithContext(ctx).
			Transaction(func(tx *gorm.DB) error {
				return tx.Raw(`UPDATE bundle_items
			SET state = 'UPLOADING'::bundle_state, updated_at = NOW()
			WHERE interaction_id IN (SELECT interaction_id
				FROM bundle_items
				WHERE state = 'PENDING'::bundle_state
				ORDER BY interaction_id ASC
				LIMIT ?
				FOR UPDATE SKIP LOCKED)
			RETURNING *`, self.Config.Bundler.PollerMaxBatchSize).
					Scan(&bundleItems).Error
			})

		if err != nil {
			if err != gorm.ErrRecordNotFound {
				self.Log.WithError(err).Error("Failed to get new interactions")
				self.monitor.GetReport().Bundler.Errors.PollerFetchError.Inc()
			}
			return nil
		}

		if len(bundleItems) > 0 {
			self.Log.WithField("count", len(bundleItems)).Debug("Polled new bundle items")
		}

		for i := range bundleItems {
			select {
			case <-self.StopChannel:
				return nil
			case self.output <- &bundleItems[i]:
			}

			// Update metrics
			self.monitor.GetReport().Bundler.State.BundlesFromSelects.Inc()
		}

		// Start another check if there can be more items to fetch
		// Skip this if another check is scheduled
		if len(bundleItems) != self.Config.Bundler.PollerMaxBatchSize {
			return nil
		}
	}
}

func (self *Poller) handleRetrying() error {
	// Repeat until there are no more interactions to fetch
	for {
		ctx, cancel := context.WithTimeout(self.Ctx, self.Config.Bundler.PollerTimeout)
		defer cancel()

		// Inserts interactions that weren't yet bundled into bundle_items table
		var bundleItems []model.BundleItem
		err := self.db.WithContext(ctx).
			Transaction(func(tx *gorm.DB) error {
				return tx.Raw(`UPDATE bundle_items
			SET state = 'UPLOADING'::bundle_state, updated_at = NOW()
			WHERE interaction_id IN (
				SELECT interaction_id
				FROM bundle_items
				WHERE state = 'UPLOADING'::bundle_state AND updated_at < NOW() - ?::interval
				ORDER BY interaction_id ASC
				LIMIT ?
				FOR UPDATE SKIP LOCKED)
			RETURNING *`, fmt.Sprintf("%d seconds", int((self.Config.Bundler.PollerRetryBundleAfter.Seconds()))), self.Config.Bundler.PollerMaxBatchSize).
					Scan(&bundleItems).Error
			})

		if err != nil {
			if err != gorm.ErrRecordNotFound {
				self.Log.WithError(err).Error("Failed to get interactions for retrying")
				self.monitor.GetReport().Bundler.Errors.PollerFetchError.Inc()
			}
			return nil
		}

		if len(bundleItems) > 0 {
			self.Log.WithField("count", len(bundleItems)).Trace("Polled bundle items for retrying")
		}

		for i := range bundleItems {
			select {
			case <-self.StopChannel:
				return nil
			case self.output <- &bundleItems[i]:
			}

			// Update metrics
			self.monitor.GetReport().Bundler.State.RetriedBundlesFromSelects.Inc()
		}

		// Start another check if there can be more items to fetch
		// Skip this if another check is scheduled
		if len(bundleItems) != self.Config.Bundler.PollerMaxBatchSize {
			return nil
		}
	}
}
