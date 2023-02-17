package bundle

import (
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

// Gets the unbundled interactions and puts them on the output channel
type InteractionMonitor struct {
	*task.Task

	notifier *Notifier
	poller   *Poller

	// Data about the interactions that need to be bundled
	BundleItems chan *model.BundleItem
}

// Sets up the task of fetching interactions to be bundled
// Collects it from two sources:
// 1. Live source of interactions that need to be bundled
// 2. Interactions that somehow wasn't sent through the notification channel. Probably because of a restart.
func NewInteractionMonitor(config *config.Config, db *gorm.DB) (self *InteractionMonitor) {
	self = new(InteractionMonitor)

	self.BundleItems = make(chan *model.BundleItem)

	self.notifier = NewNotifier(config).
		WithDB(db).
		WithOutputChannel(self.BundleItems)

	self.poller = NewPoller(config).
		WithDB(db).
		WithOutputChannel(self.BundleItems)

	self.Task = task.NewTask(config, "interaction-monitor").
		// Live source of interactions
		WithSubtask(self.notifier.Task).
		// Polled interactions
		WithSubtask(self.poller.Task).
		WithOnAfterStop(func() {
			close(self.BundleItems)
		})

	return
}
