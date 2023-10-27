package initsequencer

import (
	"gorm.io/gorm"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type NextFinishedBlock struct {
	*task.Task
	db     *gorm.DB
	Output chan *arweave.NetworkInfo
}

func NewNextFinishedBlock(config *config.Config) (self *NextFinishedBlock) {
	self = new(NextFinishedBlock)

	self.Output = make(chan *arweave.NetworkInfo)

	self.Task = task.NewTask(config, "next-finished-block").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *NextFinishedBlock) WithDB(db *gorm.DB) *NextFinishedBlock {
	self.db = db
	return self
}

func (self *NextFinishedBlock) run() (err error) {
	var state model.State
	err = self.db.WithContext(self.Ctx).
		First(&state, model.SyncedComponentInteractions).
		Error
	if err != nil {
		return
	}

	self.Output <- &arweave.NetworkInfo{
		Height: int64(state.FinishedBlockHeight),
	}

	// Wait till the context is done
	<-self.Ctx.Done()

	return
}
