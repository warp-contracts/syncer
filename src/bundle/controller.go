package bundle

import (
	"syncer/src/utils/config"
	"syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates main syncer functionalities
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)

	interactionMonitor, err := NewInteractionMonitor(config)
	if err != nil {
		return
	}
	self.Task = task.NewTask(config, "bundle-controller").
		WithSubtask(interactionMonitor.Task)
	return
}
