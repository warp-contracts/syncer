package check

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/bundlr"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates everything
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "bundle-controller")

	// SQL database
	db, err := model.NewConnection(self.CtxRunning, config)
	if err != nil {
		return
	}

	// Arweave client
	client := arweave.NewClient(self.CtxRunning, config)

	// Bundlr client
	bundlrClient := bundlr.NewClient(self.Ctx, &config.Bundlr)

	// Gets network height from WARP's GW
	networkMonitor := listener.NewNetworkMonitor(config, config.Bundler.CheckerInterval).
		WithClient(client).
		WithRequiredConfirmationBlocks(0)

		// Gets interactions that may be finalized from the db
	poller := NewPoller(config).
		WithDB(db).
		WithInputChannel(networkMonitor.Output)

	// Checks if bundlr finalized the bundle
	checker := NewChecker(config).
		WithClient(bundlrClient).
		WithInputChannel(poller.Output)

	// Updates states of the finalized
	store := NewStore(config).
		WithDB(db)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(networkMonitor.Task).
		WithSubtask(poller.Task).
		WithSubtask(checker.Task).
		WithSubtask(store.Task)
	return
}
