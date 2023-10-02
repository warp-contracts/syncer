package initsequencer

import (
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	monitor_syncer "github.com/warp-contracts/syncer/src/utils/monitoring/syncer"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
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

	client := arweave.NewClient(self.Ctx, config).
		WithTagValidator(warp.ValidateTag)

	nextFinishedBlock := NewNextFinishedBlock(config).
		WithDB(db)

	blockDownloader := listener.NewBlockDownloader(config).
		WithClient(client).
		WithMonitor(monitor).
		WithInputChannel(nextFinishedBlock.Output).
		WithBackoff(0, config.Syncer.TransactionMaxInterval).
		WithInitStartHeight(db, model.SyncedComponentInteractions)

	writer := NewWriter(config, sequencerRepoPath).
		WithSequencerRepoPath(sequencerRepoPath).
		WithDB(db).
		WithInput(blockDownloader.Output)

	self.Task = self.Task.
		WithSubtask(nextFinishedBlock.Task).
		WithSubtask(blockDownloader.Task).
		WithSubtask(writer.Task).
		WithStopChannel(writer.Output)

	return
}
