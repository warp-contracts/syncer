package interact

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	monitor_interact "github.com/warp-contracts/syncer/src/utils/monitoring/interact"
	"github.com/warp-contracts/syncer/src/utils/sequencer"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates everything
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "interact-controller")

	// SQL database
	db, err := model.NewConnection(self.Ctx, config, "gateway")
	if err != nil {
		return
	}

	// Sequencer client
	sequencerClient := sequencer.NewClient(&config.Sequencer)

	// Monitoring
	monitor := monitor_interact.NewMonitor()
	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	// Generate DataItem
	generator := NewGenerator(config).
		WithClient(sequencerClient)

	// Send interaction to Warp sequencer
	sender := NewSender(config).
		WithClient(sequencerClient).
		WithInputChannel(generator.Output).
		WithMonitor(monitor)

	// Check if database is interaction got saved to the database
	checker := NewChecker(config).
		WithDb(db).
		WithInputChannel(sender.Output).
		WithMonitor(monitor)

		// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(monitor.Task).
		WithSubtask(server.Task).
		WithSubtask(generator.Task).
		WithSubtask(sender.Task).
		WithSubtask(checker.Task)
	return
}
