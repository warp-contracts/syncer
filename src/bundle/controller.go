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

	db, err := model.NewConnection(self.Ctx, config)
	if err != nil {
		return
	}

	interactionMonitor, err := NewInteractionMonitor(config, db)
	if err != nil {
		return
	}

	self.Task.WithSubtask(interactionMonitor.Task)
	return
}
