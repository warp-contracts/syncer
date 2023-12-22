package initsequencer

import (
	"context"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	monitor_syncer "github.com/warp-contracts/syncer/src/utils/monitoring/syncer"
	"github.com/warp-contracts/syncer/src/utils/task"
	"gorm.io/gorm"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config, sequencerRepoPath string, env string) (self *Controller, err error) {
	self = new(Controller)

	self.Task = task.NewTask(config, "init-sequencer-controller")

	db, err := model.NewConnection(self.Ctx, self.Config, "syncer")
	if err != nil {
		panic(err)
	}

	monitor := monitor_syncer.NewMonitor().
		WithMaxHistorySize(30)

	client := arweave.NewClient(self.Ctx, config)

	lastSyncedBlock, err := getLastSyncedBlock(self.Ctx, db)
	if err != nil {
		panic(err)
	}

	nextBlock := NewNextBlock(config).
		WithLastSyncedBlock(lastSyncedBlock)

	blockDownloader := listener.NewBlockDownloader(config).
		WithClient(client).
		WithMonitor(monitor).
		WithInputChannel(nextBlock.Output).
		WithBackoff(0, config.Syncer.TransactionMaxInterval).
		WithHeightRange(lastSyncedBlock.FinishedBlockHeight+1, lastSyncedBlock.FinishedBlockHeight+1)

	transactionDownloader := listener.NewTransactionDownloader(config).
		WithClient(client).
		WithInputChannel(blockDownloader.Output).
		WithMonitor(monitor).
		WithBackoff(0, config.Syncer.TransactionMaxInterval).
		WithFilterInteractions(true)

	writer := NewWriter(config).
		WithSequencerRepoPath(sequencerRepoPath).
		WithEnv(env).
		WithDB(db).
		WithInput(transactionDownloader.Output).
		WithLastSyncedBlock(lastSyncedBlock)

	self.Task = self.Task.
		WithSubtask(nextBlock.Task).
		WithSubtask(blockDownloader.Task).
		WithSubtask(transactionDownloader.Task).
		WithSubtask(writer.Task).
		WithStopChannel(writer.Output)

	return
}

func getLastSyncedBlock(ctx context.Context, db *gorm.DB) (state model.State, err error) {
	err = db.WithContext(ctx).
		First(&state, model.SyncedComponentInteractions).
		Error
	return state, err
}
