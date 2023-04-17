package sync

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	monitor_syncer "syncer/src/utils/monitoring/syncer"
	"syncer/src/utils/peer_monitor"
	"syncer/src/utils/task"
	"syncer/src/utils/warp"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates main syncer functionalities
// Setups listening and storing interactions
func NewController(config *config.Config) (self *Controller, err error) {
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

		client := arweave.NewClient(self.Ctx, config).
			WithTagValidator(warp.ValidateTag)

		peerMonitor := peer_monitor.NewPeerMonitor(config).
			WithClient(client).
			WithMonitor(monitor)

		networkMonitor := listener.NewNetworkMonitor(config).
			WithClient(client).
			WithMonitor(monitor).
			WithInterval(config.ListenerPeriod).
			WithRequiredConfirmationBlocks(config.ListenerRequiredConfirmationBlocks)

		blockMonitor := listener.NewBlockMonitor(config).
			WithClient(client).
			WithInputChannel(networkMonitor.Output).
			WithMonitor(monitor).
			WithInitStartHeight(db)

		transactionMonitor := listener.NewTransactionMonitor(config).
			WithInputChannel(blockMonitor.Output).
			WithMonitor(monitor)

		store := NewStore(config).
			WithInputChannel(transactionMonitor.Output).
			WithMonitor(monitor).
			WithDB(db)

		return task.NewTask(config, "watched").
			WithSubtask(peerMonitor.Task).
			WithSubtask(store.Task).
			WithSubtask(networkMonitor.Task).
			WithSubtask(blockMonitor.Task).
			WithSubtask(transactionMonitor.Task)
	}

	watchdog := task.NewWatchdog(config).
		WithTask(watched).
		WithIsOK(func() bool {
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
		WithSubtask(watchdog.Task)

	return
}
