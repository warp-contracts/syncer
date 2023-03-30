package contract

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	monitor_contract "syncer/src/utils/monitoring/contract"
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

	self.Task = task.NewTask(config, "contract-controller")

	monitor := monitor_contract.NewMonitor().
		WithMaxHistorySize(30)

	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	watched := func() *task.Task {
		db, err := model.NewConnection(self.Ctx, self.Config, "contract")
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
			WithInterval(config.ListenerPeriod).
			WithRequiredConfirmationBlocks(config.ListenerRequiredConfirmationBlocks)

		blockDownloader := listener.NewBlockDownloader(config).
			WithClient(client).
			WithInputChannel(networkMonitor.Output).
			WithMonitor(monitor).
			WithInitStartHeight(db)

		transactionDownloader := listener.NewTransactionDownloader(config).
			WithClient(client).
			WithInputChannel(blockDownloader.Output).
			WithMonitor(monitor).
			WithFilterContracts()

		payloadToTxMapper := task.NewMapper[*listener.Payload, *arweave.Transaction](config, "transaction-to-payload").
			WithInputChannel(transactionDownloader.Output).
			WithMapFunc(func(in *listener.Payload, out chan *arweave.Transaction) (err error) {
				for _, tx := range in.Transactions {
					out <- tx
				}
				return
			})

		loader := NewLoader(config).
			WithInputChannel(payloadToTxMapper.Output).
			WithMonitor(monitor).
			WithClient(client)

		store := NewStore(config).
			WithInputChannel(loader.Output).
			WithMonitor(monitor).
			WithDB(db)

		return task.NewTask(config, "watched-contract").
			WithSubtask(peerMonitor.Task).
			WithSubtask(store.Task).
			WithSubtask(networkMonitor.Task).
			WithSubtask(blockDownloader.Task).
			WithSubtask(transactionDownloader.Task).
			WithSubtask(payloadToTxMapper.Task).
			WithSubtask(loader.Task)
	}

	watchdog := task.NewWatchdog(config).
		WithTask(watched).
		WithIsOK(func() bool {
			isOK := monitor.IsOK()
			if !isOK {
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
