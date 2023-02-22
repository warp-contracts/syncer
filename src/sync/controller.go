package sync

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitor"
	"syncer/src/utils/peer_monitor"
	"syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

// Main class that orchestrates main syncer functionalities
// Setups listening and storing interactions
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)

	self.Task = task.NewTask(config, "controller")

	client := arweave.NewClient(self.Ctx, config)

	db, err := model.NewConnection(self.Ctx, self.Config)
	if err != nil {
		return
	}

	monitor := monitor.NewMonitor()

	peerMonitor := peer_monitor.NewPeerMonitor(config).
		WithClient(client)

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

	server := NewServer(config).
		WithMonitor(monitor)

	self.Task = self.Task.
		WithSubtask(peerMonitor.Task).
		WithSubtask(store.Task).
		WithSubtask(networkMonitor.Task).
		WithSubtask(blockMonitor.Task).
		WithSubtask(transactionMonitor.Task).
		WithSubtask(server.Task)

	return
}
