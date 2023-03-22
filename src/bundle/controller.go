package bundle

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/bundlr"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	monitor_bundler "syncer/src/utils/monitoring/bundler"
	"syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// +---------------+
// |   Collector   |
// |               |
// |               |
// | +-----------+ |
// | |  Poller   | |             +----------+         +-----------+                +-----------------+
// | +-----------+ |     tx      |          |   pd    |           |  network_info  |                 |
// |               +------------>| Bundler  +-------->| Confirmer |<-------------- | Network Monitor |
// | +-----------+ |             |          |         |           |                |                 |
// | |  Notifier | |             +----------+         +-----------+                +-----------------+
// | +-----------+ |
// |               |
// +---------------+
// Main class that orchestrates main syncer functionalities
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "bundle-controller")

	// SQL database
	db, err := model.NewConnection(self.Ctx, config, "bundler")
	if err != nil {
		return
	}

	// Arweave client
	arweaveClient := arweave.NewClient(self.Ctx, config)

	// Bundlr client
	bundlrClient := bundlr.NewClient(self.Ctx, &config.Bundlr)

	// Monitoring
	monitor := monitor_bundler.NewMonitor()
	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	// Gets interactions to bundle from the database
	collector := NewCollector(config, db).
		WithMonitor(monitor)

	// Monitors latest Arweave network block height
	networkMonitor := listener.NewNetworkMonitor(config).
		WithClient(arweaveClient).
		WithMonitor(monitor).
		WithInterval(config.ListenerPeriod).
		WithRequiredConfirmationBlocks(0)

	// Sends interactions to bundlr.network
	bundler := NewBundler(config, db).
		WithInputChannel(collector.Output).
		WithMonitor(monitor).
		WithClient(bundlrClient)

	// Confirmer periodically updates the state of the bundled interactions
	confirmer := NewConfirmer(config).
		WithDB(db).
		WithMonitor(monitor).
		WithNetworkMonitor(networkMonitor).
		WithInputChannel(bundler.Output)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(confirmer.Task).
		WithSubtask(bundler.Task).
		WithSubtask(monitor.Task).
		WithSubtask(networkMonitor.Task).
		WithSubtask(server.Task).
		WithSubtask(collector.Task)
	return
}
