package relay

import (
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	monitor_relayer "github.com/warp-contracts/syncer/src/utils/monitoring/relayer"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "relayer")

	// Monitoring
	monitor := monitor_relayer.NewMonitor(config).
		WithMaxHistorySize(30)
	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	watched := func() *task.Task {
		// SQL database
		db, err := model.NewConnection(self.Ctx, config, "relayer")
		if err != nil {
			panic(err)
		}

		// Arweave client
		client := arweave.NewClient(self.Ctx, config).
			WithTagValidator(warp.ValidateTag)

		// Sequencer/Cosmos client
		sequencerClient, err := rpchttp.New(config.Relayer.SequencerUrl, "/websocket")
		if err != nil {
			panic(err)
		}

		// Events from Warp's sequencer
		streamer := NewStreamer(config).
			WithClient(sequencerClient).
			WithMonitor(monitor)
		streamer.Resume()

		// Forwards blocks from Streamer, but fills in the gaps.
		source := NewSource(config).
			WithDB(db).
			WithMonitor(monitor).
			WithClient(sequencerClient).
			WithInputChannel(streamer.Output)

		// Parses blocks into payload
		parser := NewParser(config).
			WithMonitor(monitor).
			WithInputChannel(source.Output)

		// Monitor current network height (output is disabled)
		networkMonitor := listener.NewNetworkMonitor(config).
			WithClient(client).
			WithMonitor(monitor).
			WithInterval(config.NetworkMonitor.Period).
			WithEnableOutput(false)

		// Fill in Arweave blocks
		blockDownloader := NewOneBlockDownloader(config).
			WithMonitor(monitor).
			WithClient(client).
			WithInputChannel(parser.Output)

		// Download transactions from Arweave
		transactionOrchestrator := NewTransactionOrchestrator(config).
			WithInputChannel(blockDownloader.Output)

		transactionDownloader := listener.NewTransactionDownloader(config).
			WithClient(client).
			WithInputChannel(transactionOrchestrator.TransactionOutput).
			WithMonitor(monitor).
			WithBackoff(0, config.Relayer.ArweaveBlockDownloadMaxElapsedTime).
			WithFilterInteractions()

		transactionOrchestrator.WithTransactionInput(transactionDownloader.Output)

		// Parse arweave transactions into interactions
		arweaveParser := NewArweaveParser(config).
			WithInputChannel(transactionOrchestrator.Output)

		// Store blocks in the database, in batches
		store := NewStore(config).
			WithInputChannel(arweaveParser.Output).
			WithMonitor(monitor).
			WithDB(db)

		return task.NewTask(config, "watched").
			WithSubtask(source.Task).
			WithSubtask(parser.Task).
			WithSubtask(networkMonitor.Task).
			WithSubtask(blockDownloader.Task).
			WithSubtask(transactionDownloader.Task).
			WithSubtask(transactionOrchestrator.Task).
			WithSubtask(arweaveParser.Task).
			WithSubtask(store.Task).
			WithSubtask(streamer.Task)
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

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(monitor.Task).
		WithSubtask(server.Task).
		WithSubtask(watchdog.Task)

	return
}
