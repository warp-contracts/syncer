package load

import (
	"syncer/src/utils/config"
	"syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates functionalities
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "loader-controller")

	// SQL database
	// db, err := model.NewConnection(self.Ctx, config, "bundle-load-test")
	// if err != nil {
	// 	return
	// }

	// Generates bundles
	generator := NewGenerator(config)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(generator.Task)
		// WithSubtask(store.Task)
	return
}
