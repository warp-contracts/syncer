package fixsortkey

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates main syncer functionalities
// Setups listening and storing interactions
func NewController(config *config.Config, startBlockHeight, stopBlockHeight uint64) (self *Controller, err error) {
	self = new(Controller)

	self.Task = task.NewTask(config, "fixsortkey-controller")

	db, err := model.NewConnection(self.Ctx, self.Config, "fix-sort-key")
	if err != nil {
		panic(err)
	}

	sequencer := NewSequencer(self.Config).
		WithStart(startBlockHeight).
		WithStop(stopBlockHeight)

	processor := NewProcessor(self.Config).
		WithInputChannel(sequencer.Output).
		WithDB(db)

	self.Task = self.Task.
		WithSubtask(processor.Task).
		WithSubtask(sequencer.Task)

	return
}
