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
	"github.com/warp-contracts/syncer/src/utils/peer_monitor"
	"github.com/warp-contracts/syncer/src/utils/task"
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
		client := arweave.NewClient(self.Ctx, config)

		// Monitor current network height (output is disabled)
		networkMonitor := listener.NewNetworkMonitor(config).
			WithClient(client).
			WithMonitor(monitor).
			WithInterval(config.NetworkMonitor.Period).
			WithEnableOutput(false)

		peerMonitor := peer_monitor.NewPeerMonitor(config).
			WithClient(client).
			WithMonitor(monitor)

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

		// Decodes messages, creates payload
		decoder := NewDecoder(config).
			WithMonitor(monitor).
			WithInputChannel(source.Output)

		// Fills in arweave block info in the payload
		msgArweaveBlockParser := NewMsgArweaveBlockParser(config).
			WithMonitor(monitor).
			WithInputChannel(decoder.Output)

		lastArweaveBlockProvider := NewLastArweaveBlockProvider(config).
			WithInputChannel(msgArweaveBlockParser.Output).
			WithClient(sequencerClient).
			WithDecoder(decoder).
			WithMonitor(monitor)

		// Parses blocks into payload
		msgDataItemParser := NewMsgDataItemParser(config).
			WithMonitor(monitor).
			WithInputChannel(lastArweaveBlockProvider.Output)

		// Fill in Arweave blocks
		blockDownloader := NewOneBlockDownloader(config).
			WithMonitor(monitor).
			WithClient(client).
			WithInputChannel(msgDataItemParser.Output)

		// Download transactions from Arweave, but only those specified by the sequencer
		transactionDownloader := NewTransactionDownloader(config).
			WithMonitor(monitor).
			WithClient(client).
			WithInputChannel(blockDownloader.Output)

		// Parse arweave transactions into interactions
		arweaveParser := NewArweaveParser(config).
			WithMonitor(monitor).
			WithInputChannel(transactionDownloader.Output)

		// Store blocks in the database, in batches
		store := NewStore(config).
			WithInputChannel(arweaveParser.Output).
			WithMonitor(monitor).
			WithIsReplacing(true).
			WithDB(db)

		return task.NewTask(config, "watched").
			WithSubtask(source.Task).
			WithSubtask(decoder.Task).
			WithSubtask(msgArweaveBlockParser.Task).
			WithSubtask(lastArweaveBlockProvider.Task).
			WithSubtask(msgDataItemParser.Task).
			WithSubtask(networkMonitor.Task).
			WithSubtask(blockDownloader.Task).
			WithSubtask(transactionDownloader.Task).
			WithSubtask(arweaveParser.Task).
			WithSubtask(store.Task).
			WithSubtask(streamer.Task).
			WithSubtask(peerMonitor.Task)
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
