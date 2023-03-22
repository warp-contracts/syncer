package bundle

import (
	"encoding/json"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/notify"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

// Gets a live stream of unbundled intearctions, parses them and puts them on the output channel
type Notifier struct {
	*task.Task
	db *gorm.DB

	streamer *notify.Streamer
	monitor  monitoring.Monitor

	// Data about the interactions that need to be bundled
	output chan *model.BundleItem
}

func NewNotifier(config *config.Config) (self *Notifier) {
	self = new(Notifier)

	self.streamer = notify.NewStreamer(config, "bundler-notifier").
		WithNotificationChannelName("bundle_items_pending").
		WithCapacity(10)

	self.Task = task.NewTask(config, "notifier").
		// Live source of interactions that need to be bundled
		WithSubtask(self.streamer.Task).
		// Interactions that somehow wasn't sent through the notification channel. Probably because of a restart.
		WithSubtaskFunc(self.run).
		// Workers unmarshal big JSON messages and optionally fetch data from the database if the messages wuldn't fit in the notification channel
		WithWorkerPool(config.Bundler.NotifierWorkerPoolSize, config.Bundler.NotifierWorkerQueueSize)

	return
}

func (self *Notifier) WithDB(db *gorm.DB) *Notifier {
	self.db = db
	return self
}

func (self *Notifier) WithMonitor(monitor monitoring.Monitor) *Notifier {
	self.monitor = monitor
	return self
}

func (self *Notifier) WithOutputChannel(bundleItems chan *model.BundleItem) *Notifier {
	self.output = bundleItems
	return self
}

func (self *Notifier) run() error {
	for {
		select {
		case <-self.StopChannel:
			self.Log.Debug("Stop passing interactions from notification")
			return nil
		case msg, ok := <-self.streamer.Output:
			if !ok {
				self.Log.Info("Notification streamer channel closed")
				return nil
			}
			self.SubmitToWorker(func() {
				var notification model.BundleItemNotification
				err := json.Unmarshal([]byte(msg), &notification)
				if err != nil {
					self.Log.WithError(err).Error("Failed to unmarshal notification")
					return
				}

				bundleItem := model.BundleItem{
					InteractionID: notification.InteractionID,
				}
				if notification.Transaction != nil {
					// FIXME: This copies a lot of data
					bundleItem.Transaction = *notification.Transaction
					bundleItem.Tags = *notification.Tags
				} else {
					// Transaction was too big to fit into the notification channel
					// Only id is there, we need to fetch the rest of the data from the database
					err = self.db.WithContext(self.Ctx).
						Model(&model.BundleItem{}).
						Select("transaction").
						Where("interaction_id = ?", notification.InteractionID).
						Scan(&bundleItem).
						Error
					if err != nil {
						self.Log.WithError(err).Error("Failed to get bundle item")
						return
					}
				}

				select {
				case <-self.StopChannel:
					return
				case self.output <- &bundleItem:
				}

				// Update metrics
				self.monitor.GetReport().Bundler.State.BundlesFromNotifications.Inc()
			})
		}
	}
}
