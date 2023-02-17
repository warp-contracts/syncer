package bundle

import (
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates main syncer functionalities
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "bundle-controller")

	// SQL database
	db, err := model.NewConnection(self.CtxRunning, config)
	if err != nil {
		return
	}

	// Gets interactions to bundle from the database
	collector := NewCollector(config, db)

	// Sends interactions to bundlr.network
	bundler := NewBundler(config, db).
		WithInputChannel(collector.BundleItems)
	if err != nil {
		return
	}

	// Confirmer periodically updates the state of the bundled interactions
	confirmer := NewConfirmer(config).
		WithDB(db).
		WithInputChannel(bundler.Bundled)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(confirmer.Task).
		WithSubtask(bundler.Task).
		WithSubtask(collector.Task)
	return
}
