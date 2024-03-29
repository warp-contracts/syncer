package sync

import (
	"time"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	monitor_syncer "github.com/warp-contracts/syncer/src/utils/monitoring/syncer"
	"github.com/warp-contracts/syncer/src/utils/peer_monitor"
	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates main syncer functionalities
// Setups listening and storing interactions
func NewController(config *config.Config, startBlockHeight, stopBlockHeight uint64, replaceExisting bool) (self *Controller, err error) {
	self = new(Controller)

	self.Task = task.NewTask(config, "controller")

	monitor := monitor_syncer.NewMonitor().
		WithMaxHistorySize(30)

	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	watched := func() *task.Task {
		db, err := model.NewConnection(self.Ctx, self.Config, "syncer")
		if err != nil {
			panic(err)
		}

		client := arweave.NewClient(self.Ctx, config)

		peerMonitor := peer_monitor.NewPeerMonitor(config).
			WithClient(client).
			WithMonitor(monitor)

		networkMonitor := listener.NewNetworkMonitor(config).
			WithClient(client).
			WithMonitor(monitor).
			WithInterval(config.NetworkMonitor.Period).
			WithRequiredConfirmationBlocks(config.NetworkMonitor.RequiredConfirmationBlocks)

		blockDownloader := listener.NewBlockDownloader(config).
			WithClient(client).
			WithInputChannel(networkMonitor.Output).
			WithMonitor(monitor).
			WithBackoff(0, config.Syncer.TransactionMaxInterval)

		if startBlockHeight > 0 && stopBlockHeight > 0 {
			// Sync only a range of blocks
			blockDownloader = blockDownloader.WithHeightRange(startBlockHeight, stopBlockHeight)
		} else if startBlockHeight <= 0 && stopBlockHeight > 0 {
			// Sync normally, but stop at a given height
			blockDownloader = blockDownloader.WithStopHeight(db, stopBlockHeight, model.SyncedComponentInteractions)
		} else {
			// By default sync using the height saved in the db and never stop
			blockDownloader = blockDownloader.WithInitStartHeight(db, model.SyncedComponentInteractions)
		}

		transactionDownloader := listener.NewTransactionDownloader(config).
			WithClient(client).
			WithInputChannel(blockDownloader.Output).
			WithMonitor(monitor).
			WithBackoff(0, config.Syncer.TransactionMaxInterval).
			WithFilterInteractions(false /* input from tx's data is disabled */)

		parser := NewParser(config).
			WithInputChannel(transactionDownloader.Output).
			WithMonitor(monitor)

		store := NewStore(config).
			WithInputChannel(parser.Output).
			WithMonitor(monitor).
			WithReplaceExistingData(replaceExisting).
			WithDB(db)

		return task.NewTask(config, "watched").
			WithSubtask(peerMonitor.Task).
			WithSubtask(networkMonitor.Task).
			WithSubtask(blockDownloader.Task).
			WithSubtask(transactionDownloader.Task).
			WithSubtask(parser.Task).
			WithSubtask(store.Task)
	}

	watchdog := task.NewWatchdog(config).
		WithTask(watched).
		WithIsOK(30*time.Second, func() bool {
			isOK := monitor.IsOK()
			if !isOK {
				monitor.Clear()
				monitor.GetReport().Run.Errors.NumWatchdogRestarts.Inc()
			}
			return isOK
		})

	self.Task = self.Task.
		WithSubtask(monitor.Task).
		WithSubtask(server.Task).
		WithConditionalSubtask(config.Syncer.Enabled, watchdog.Task)

	return
}
