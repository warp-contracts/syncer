package initsequencer

import (
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type NextBlock struct {
	*task.Task
	lastSyncedBlock model.State
	Output          chan *arweave.NetworkInfo
}

func NewNextBlock(config *config.Config) (self *NextBlock) {
	self = new(NextBlock)

	self.Output = make(chan *arweave.NetworkInfo)

	self.Task = task.NewTask(config, "next-block").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *NextBlock) WithLastSyncedBlock(lastSyncedBlock model.State) *NextBlock {
	self.lastSyncedBlock = lastSyncedBlock
	return self
}

func (self *NextBlock) run() (err error) {
	self.Output <- &arweave.NetworkInfo{
		Height: int64(self.lastSyncedBlock.FinishedBlockHeight) + 1,
	}

	// Wait till the context is done
	<-self.Ctx.Done()

	return
}
