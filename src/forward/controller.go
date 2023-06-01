package forward

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	monitor_forwarder "github.com/warp-contracts/syncer/src/utils/monitoring/forwarder"
	"github.com/warp-contracts/syncer/src/utils/task"
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

	// Fetches L1 interactions from the DB every time the block height changes
	fetcher := NewFetcher(config).
		WithDB(db).
		WithMonitor(monitor).
		WithInputChannel(sequencer.Output)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(sequencer.Task).
		WithSubtask(monitor.Task).
		WithSubtask(fetcher.Task).
		WithSubtask(server.Task)
	return
}
