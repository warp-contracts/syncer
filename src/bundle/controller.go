package bundle

import (
	"syncer/src/utils/bundlr"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
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
// | |  Poller   | |             +----------+         +-----------+
// | +-----------+ |     tx      |          |   pd    |           |
// |               +------------>| Bundler  +-------->| Confirmer |
// | +-----------+ |             |          |         |           |
// | |  Notifier | |             +----------+         +-----------+
// | +-----------+ |
// |               |
// +---------------+
// Main class that orchestrates main syncer functionalities
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "bundle-controller")

	// SQL database
	db, err := model.NewConnection(self.Ctx, config)
	if err != nil {
		return
	}

	// Bundlr client
	bundlrClient := bundlr.NewClient(self.Ctx, &config.Bundlr)

	// Gets interactions to bundle from the database
	collector := NewCollector(config, db)

	// Sends interactions to bundlr.network
	bundler := NewBundler(config, db).
		WithInputChannel(collector.Output).
		WithClient(bundlrClient)

	// Confirmer periodically updates the state of the bundled interactions
	confirmer := NewConfirmer(config).
		WithDB(db).
		WithInputChannel(bundler.Output)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(confirmer.Task).
		WithSubtask(bundler.Task).
		WithSubtask(collector.Task)
	return
}
