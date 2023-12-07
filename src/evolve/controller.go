package evolve

import (
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "evolve-controller")

	// SQL database
	db, err := model.NewConnection(self.Ctx, config, "evolve")
	if err != nil {
		return
	}

	// Arweave client
	client := arweave.NewClient(self.Ctx, config)


	// Gets new contract sources from the database
	poller := NewPoller(config).
			WithDB(db)

	// Downloads source transaction and loads its metadata
	downloader := NewDownloader(config).
			WithInputChannel(poller.Output).
			WithClient(client)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(poller.Task).
		WithSubtask(downloader.Task)
	return
}
