package send

import (
	"context"
	"fmt"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets pending data items and puts them on the output channel
// Gets data items that somehow didn't get sent through the notification channel.
type Poller struct {
	*task.Task

	db      *gorm.DB
	monitor monitoring.Monitor
	output  chan *model.DataItem
}

func NewPoller(config *config.Config) (self *Poller) {
	self = new(Poller)

	self.Task = task.NewTask(config, "poller").
		WithRepeatedSubtaskFunc(config.Sender.PollerInterval, self.handleNew).
		WithRepeatedSubtaskFunc(config.Sender.PollerInterval, self.handleRetrying)

	return
}

func (self *Poller) WithDB(db *gorm.DB) *Poller {
	self.db = db
	return self
}

func (self *Poller) WithOutputChannel(v chan *model.DataItem) *Poller {
	self.output = v
	return self
}

func (self *Poller) WithMonitor(monitor monitoring.Monitor) *Poller {
	self.monitor = monitor
	return self
}

func (self *Poller) handleNew() (repeat bool, err error) {
	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.Sender.PollerTimeout)
	defer cancel()

	// Gets new data items
	var dataItems []model.DataItem
	err = self.db.WithContext(ctx).
		Raw(`UPDATE data_items
			SET state = 'UPLOADING'::bundle_state, updated_at = NOW()
			WHERE data_item_id IN (SELECT data_item_id
				FROM data_items
				WHERE state = 'PENDING'::bundle_state
				ORDER BY updated_at ASC
				LIMIT ?
				FOR UPDATE SKIP LOCKED)
			RETURNING *`, self.Config.Sender.PollerMaxBatchSize).
		Scan(&dataItems).Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			self.Log.WithError(err).Error("Failed to get new data items")
			self.monitor.GetReport().Sender.Errors.PollerFetchError.Inc()
		}
		err = nil
		return
	}

	if len(dataItems) > 0 {
		self.Log.WithField("count", len(dataItems)).Debug("Polled new data items")
	}

	for i := range dataItems {
		select {
		case <-self.Ctx.Done():
			return
		case self.output <- &dataItems[i]:
		}

		// Update metrics
		self.monitor.GetReport().Sender.State.BundlesFromSelects.Inc()
	}

	// Start another check if there can be more items to fetch
	// Skip this if another check is scheduled
	if len(dataItems) != self.Config.Sender.PollerMaxBatchSize {
		return
	}

	// Repeat right away
	repeat = true
	return
}

func (self *Poller) handleRetrying() (repeat bool, err error) {

	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.Sender.PollerTimeout)
	defer cancel()

	var dataItems []model.DataItem
	err = self.db.WithContext(ctx).
		Raw(`UPDATE data_items
	SET state = 'UPLOADING'::bundle_state, updated_at = NOW()
	WHERE data_item_id IN (
		SELECT data_item_id
		FROM data_items
		WHERE state = 'UPLOADING'::bundle_state AND updated_at < NOW() - ?::interval
		ORDER BY updated_at ASC
		LIMIT ?
		FOR UPDATE SKIP LOCKED)
	RETURNING *`, fmt.Sprintf("%d seconds", int((self.Config.Sender.PollerRetryBundleAfter.Seconds()))), self.Config.Sender.PollerMaxBatchSize).
		Scan(&dataItems).
		Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			self.Log.WithError(err).Error("Failed to get data items for retrying")
			self.monitor.GetReport().Sender.Errors.PollerFetchError.Inc()
		}
		err = nil
		return
	}

	if len(dataItems) > 0 {
		self.Log.WithField("count", len(dataItems)).Trace("Polled data items for retrying sending to bundling service")
	}

	for i := range dataItems {
		select {
		case <-self.Ctx.Done():
			return
		case self.output <- &dataItems[i]:
		}

		// Update metrics
		self.monitor.GetReport().Sender.State.RetriedBundlesFromSelects.Inc()
	}

	// Start another check if there can be more items to fetch
	// Skip this if another check is scheduled
	if len(dataItems) != self.Config.Sender.PollerMaxBatchSize {
		return
	}

	repeat = true
	return
}
