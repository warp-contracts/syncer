package contract

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	monitor_contract "syncer/src/utils/monitoring/contract"
	"syncer/src/utils/peer_monitor"
	"syncer/src/utils/publisher"
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

		blockDownloader := listener.NewBlockDownloader(config).
			WithClient(client).
			WithInputChannel(networkMonitor.Output).
			WithMonitor(monitor).
			WithInitStartHeight(db, listener.ComponentContract)

		transactionDownloader := listener.NewTransactionDownloader(config).
			WithClient(client).
			WithInputChannel(blockDownloader.Output).
			WithMonitor(monitor).
			WithBackoff(config.Contract.TransactionMaxElapsedTime, config.Contract.TransactionMaxInterval).
			WithFilterContracts()

		loader := NewLoader(config).
			WithInputChannel(transactionDownloader.Output).
			WithMonitor(monitor).
			WithClient(client)

		store := NewStore(config).
			WithInputChannel(loader.Output).
			WithMonitor(monitor).
			WithDB(db)

		mapper := NewMapper(config).
			WithInputChannel(store.Output)

		redisPublisher := publisher.NewRedisPublisher[*model.ContractNotification](config, "contract-redis-publisher").
			WithChannelName(config.Contract.PublisherRedisChannelName).
			WithInputChannel(mapper.Output)

		return task.NewTask(config, "watched-contract").
			WithSubtask(peerMonitor.Task).
			WithSubtask(store.Task).
			WithSubtask(networkMonitor.Task).
			WithSubtask(blockDownloader.Task).
			WithSubtask(transactionDownloader.Task).
			WithSubtask(mapper.Task).
			WithSubtask(redisPublisher.Task).
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
