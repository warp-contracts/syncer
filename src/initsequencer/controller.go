package initsequencer

import (
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	monitor_syncer "github.com/warp-contracts/syncer/src/utils/monitoring/syncer"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config, sequencerRepoPath string) (self *Controller, err error) {
	self = new(Controller)

	self.Task = task.NewTask(config, "init-sequencer-controller")

	db, err := model.NewConnection(self.Ctx, self.Config, "syncer")
	if err != nil {
		panic(err)
	}

	monitor := monitor_syncer.NewMonitor().
		WithMaxHistorySize(30)

	client := arweave.NewClient(self.Ctx, config)

	finishedBlock := NewFinishedBlock(config).
		WithDB(db)

	lastSyncedBlock, err := finishedBlock.getLastSyncedBlock()
	if err != nil {
		panic(err)
	}

	blockDownloader := listener.NewBlockDownloader(config).
		WithClient(client).
		WithMonitor(monitor).
		WithInputChannel(finishedBlock.Output).
		WithBackoff(0, config.Syncer.TransactionMaxInterval).
		WithHeightRange(lastSyncedBlock.FinishedBlockHeight, lastSyncedBlock.FinishedBlockHeight)

	writer := NewWriter(config).
		WithSequencerRepoPath(sequencerRepoPath).
		WithDB(db).
		WithInput(blockDownloader.Output)

	self.Task = self.Task.
		WithSubtask(finishedBlock.Task).
		WithSubtask(blockDownloader.Task).
		WithSubtask(writer.Task).
		WithStopChannel(writer.Output)

	return
}
