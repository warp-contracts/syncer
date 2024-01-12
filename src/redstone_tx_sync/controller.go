package redstone_tx_sync

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	monitor_redstone_tx_syncer "github.com/warp-contracts/syncer/src/utils/monitoring/redstone_tx_syncer"

	"github.com/warp-contracts/syncer/src/utils/sequencer"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "redstone_tx_syncer")

	// SQL database
	db, err := model.NewConnection(self.Ctx, config, "redstone_tx_syncer")
	if err != nil {
		return
	}

	// Monitoring
	monitor := monitor_redstone_tx_syncer.NewMonitor()
	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	// Sequencer client
	sequencerClient := sequencer.NewClient(&config.Sequencer)

	// Eth client
	ethClient, err := eth.GetEthClient(self.Log, eth.Avax)
	if err != nil {
		self.Log.WithError(err).Error("Could not get ETH client")
		return
	}

	// Downloads new blocks
	blockDownloader := NewBlockDownloader(config).
		WithInitStartBlockHeight(db).
		WithMonitor(monitor).
		WithEthClient(ethClient)

	// Checks wether block's transactions contain Redstone data and if so - writes interaction to Warpy
	syncer := NewSyncer(config).
		WithMonitor(monitor).
		WithInputChannel(blockDownloader.Output).
		WithSequencerClient(sequencerClient)

	// Periodically stores last synced block height in the database
	store := NewStore(config).
		WithInputChannel(syncer.Output).
		WithMonitor(monitor).
		WithDb(db)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(blockDownloader.Task).
		WithSubtask(syncer.Task).
		WithSubtask(store.Task).
		WithSubtask(monitor.Task).
		WithSubtask(server.Task)
	return
}
