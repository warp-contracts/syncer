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
	db, err := model.NewConnection(self.Ctx, config)
	if err != nil {
		return
	}

	// Gets interactions to bundle from the database
	interactionMonitor := NewInteractionMonitor(config, db)

	// Sends interactions to bundler
	bundlerManager := NewBundlerManager(config, db).
		WithInputChannel(interactionMonitor.BundleItems)
	if err != nil {
		return
	}
	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(interactionMonitor.Task).
		WithSubtask(bundlerManager.Task)
	return
}
