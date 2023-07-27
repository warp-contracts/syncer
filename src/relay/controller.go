package relay

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	monitor_relayer "github.com/warp-contracts/syncer/src/utils/monitoring/relayer"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "relayer")

	// SQL database
	// db, err := model.NewConnection(self.Ctx, config, "relayer")
	// if err != nil {
	// 	return
	// }

	// Monitoring
	monitor := monitor_relayer.NewMonitor(config)
	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	// Events from Warp's sequencer
	streamer := NewStreamer(config).
		WithMonitor(monitor)
	streamer.Resume()

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(monitor.Task).
		WithSubtask(server.Task).
		WithSubtask(streamer.Task)
	return
}
