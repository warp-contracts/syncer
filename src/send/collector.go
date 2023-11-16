package send

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Gets data items from 2 sources
type Collector struct {
	*task.Task

	notifier *Notifier
	poller   *Poller

	// Data items to be sent out
	Output chan *model.DataItem
}

func NewCollector(config *config.Config, db *gorm.DB) (self *Collector) {
	self = new(Collector)

	self.Output = make(chan *model.DataItem, 100)

	self.notifier = NewNotifier(config).
		WithDB(db).
		WithOutputChannel(self.Output)

	self.poller = NewPoller(config).
		WithDB(db).
		WithOutputChannel(self.Output)

	self.Task = task.NewTask(config, "collector").
		WithConditionalSubtask(!config.Sender.NotifierDisabled, self.notifier.Task).
		WithConditionalSubtask(!config.Sender.PollerDisabled, self.poller.Task).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Collector) WithMonitor(monitor monitoring.Monitor) *Collector {
	self.notifier.WithMonitor(monitor)
	self.poller.WithMonitor(monitor)
	return self
}
