package initsequencer

import (
	"gorm.io/gorm"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type FinishedBlock struct {
	*task.Task
	db     *gorm.DB
	Output chan *arweave.NetworkInfo
}

func NewFinishedBlock(config *config.Config) (self *FinishedBlock) {
	self = new(FinishedBlock)

	self.Output = make(chan *arweave.NetworkInfo)

	self.Task = task.NewTask(config, "next-finished-block").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *FinishedBlock) WithDB(db *gorm.DB) *FinishedBlock {
	self.db = db
	return self
}

func (self *FinishedBlock) run() (err error) {
	state, err := self.getLastSyncedBlock()
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

func (self *FinishedBlock) getLastSyncedBlock() (state model.State, err error) {
	err = self.db.WithContext(self.Ctx).
		First(&state, model.SyncedComponentInteractions).
		Error
	return state, err
}
