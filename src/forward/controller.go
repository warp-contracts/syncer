package forward

import (
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	monitor_forwarder "syncer/src/utils/monitoring/forwarder"
	"syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "forwarder")

	// SQL database
	db, err := model.NewConnection(self.Ctx, config, "forwarder")
	if err != nil {
		return
	}

	// Monitoring
	monitor := monitor_forwarder.NewMonitor()
	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	// Block height changes from sequencer
	sequencer := NewSequencer(config).
		WithMonitor(monitor).
		WithInitStartHeight(db)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(sequencer.Task).
		WithSubtask(monitor.Task).
		WithSubtask(server.Task)
	return
}
