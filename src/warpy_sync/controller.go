package warpy_sync

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/eth"
	"github.com/warp-contracts/syncer/src/utils/files"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	monitor_warpy_syncer "github.com/warp-contracts/syncer/src/utils/monitoring/warpy_syncer"
	"github.com/warp-contracts/syncer/src/utils/sequencer"

	"github.com/warp-contracts/syncer/src/utils/task"
)

type Controller struct {
	*task.Task
}

func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.Task = task.NewTask(config, "warpy_syncer")

	// SQL database
	db, err := model.NewConnection(self.Ctx, config, "warpy_syncer")
	if err != nil {
		return
	}

	// Monitoring
	monitor := monitor_warpy_syncer.NewMonitor()
	server := monitoring.NewServer(config).
		WithMonitor(monitor)

	// Sequencer client
	sequencerClient := sequencer.NewClient(&config.Sequencer)

	// Eth client
	ethClient, err := eth.GetEthClient(self.Log, config.WarpySyncer.SyncerChain)
	if err != nil {
		self.Log.WithError(err).Error("Could not get ETH client")
		return
	}

	// Synced component based on chosen chain
	var syncedComponent model.SyncedComponent
	switch config.WarpySyncer.SyncerChain {
	case eth.Avax:
		syncedComponent = model.SyncedComponentWarpySyncerAvax
	case eth.Arbitrum:
		syncedComponent = model.SyncedComponentWarpySyncerArbitrum
	case eth.Mode:
		syncedComponent = model.SyncedComponentWarpySyncerMode
	case eth.Manta:
		syncedComponent = model.SyncedComponentWarpySyncerManta
	case eth.Bsc:
		syncedComponent = model.SyncedComponentWarpySyncerBsc
	default:
		err = errors.New("synced component not recognized")
	}
	if err != nil {
		self.Log.WithError(err).Error("Failed to get synced component")
		return
	}

	// Downloads new blocks
	blockDownloader := NewBlockDownloader(config).
		WithInitStartBlockHeight(db, syncedComponent).
		WithMonitor(monitor).
		WithEthClient(ethClient)

	// Syncing tasks based on chosen protocol
	var syncerTask *task.Task
	var pollerTask *task.Task
	var assetsCalculatorTask *task.Task
	var writerTask *task.Task
	var StoreDepositTask *task.Task
	var syncerOutput chan *LastSyncedBlockPayload
	var addressesJoined string

	switch config.WarpySyncer.SyncerProtocol {
	case eth.Delta:
		// Checks wether block's transactions contain Redstone data and if so - writes interaction to Warpy
		syncer := NewSyncerDelta(config).
			WithMonitor(monitor).
			WithInputChannel(blockDownloader.Output)

		// Writes interaction to Warpy
		writer := NewWriter(config).
			WithInputChannel(syncer.OutputInteractionPayload).
			WithMonitor(monitor).
			WithSequencerClient(sequencerClient)

		writerTask = writer.Task
		syncerTask = syncer.Task
		syncerOutput = syncer.Output
	case eth.Sommelier, eth.LayerBank, eth.Pendle, eth.Venus:
		var contractAbi map[string]*abi.ABI
		contractAbi = make(map[string]*abi.ABI)

		abiSource := config.WarpySyncer.SyncerProtocol.GetAbi()

		for _, syncerDepositContractId := range config.WarpySyncer.SyncerDepositContractIds {
			if abiSource == "direct" {
				contractAbi[syncerDepositContractId], err = eth.GetContractABI(
					syncerDepositContractId,
					config.WarpySyncer.SyncerApiKey,
					config.WarpySyncer.SyncerChain)
			} else if abiSource != "" {
				contractAbi[syncerDepositContractId], err = eth.GetContractABIFromFile(abiSource)
			} else {
				err = errors.New("protocol not recognized")
			}
		}

		// to be removed in prod
		if config.WarpySyncer.SyncerProtocol == eth.Venus {
			pwd, _ := os.Getwd()
			records := files.ReadCsvFile(fmt.Sprintf("%s/src/warpy_sync/files/testers.csv", pwd))

			addresses := make([]string, len(records))
			for i := range records {
				addresses[i] = records[i][1]
			}
			addressesJoined = strings.Join(addresses[:], "|")
		}

		// Checks wether block's transactions contain specific transactions
		syncer := NewSyncerDeposit(config).
			WithMonitor(monitor).
			WithInputChannel(blockDownloader.Output).
			WithContractAbi(contractAbi).
			WithDb(db)

		blockDownloader.WithPollerCron()

		// Polls records from db
		poller := NewPollerDeposit(config).
			WithDB(db).
			WithMonitor(monitor).
			WithInputChannel(blockDownloader.OutputPollTxs).
			WithAddressesToPoll(addressesJoined)

		// Writes interaction to Warpy based on the records from the poller
		writer := NewWriter(config).
			WithInputChannel(poller.Output).
			WithMonitor(monitor).
			WithSequencerClient(sequencerClient)

		assetsCalculator := NewAssetsCalculator(config).
			WithMonitor(monitor).
			WithEthClient(ethClient).
			WithContractAbi(contractAbi).
			WithInputChannel(syncer.OutputTransactionPayload)

		StoreDeposit := NewStoreDeposit(config).
			WithDB(db).
			WithMonitor(monitor).
			WithInputChannel(assetsCalculator.Output)

		pollerTask = poller.Task
		syncerTask = syncer.Task
		syncerOutput = syncer.Output
		assetsCalculatorTask = assetsCalculator.Task
		writerTask = writer.Task
		StoreDepositTask = StoreDeposit.Task
	default:
		self.Log.WithError(err).Error("ETH Protocol not recognized")
		return
	}

	if err != nil {
		self.Log.WithError(err).Error("Could not get contract Abi")
		return
	}

	// Periodically stores last synced block height in the database
	store := NewStore(config).
		WithInputChannel(syncerOutput).
		WithMonitor(monitor).
		WithDb(db).
		WithSyncedComponent(syncedComponent)

	// Setup everything, will start upon calling Controller.Start()
	self.Task.
		WithSubtask(blockDownloader.Task).
		WithSubtask(syncerTask).
		WithSubtask(store.Task).
		WithSubtask(monitor.Task).
		WithSubtask(server.Task).
		WithConditionalSubtask(assetsCalculatorTask.Name != "", assetsCalculatorTask).
		WithConditionalSubtask(pollerTask.Name != "", pollerTask).
		WithSubtask(writerTask).
		WithSubtask(StoreDepositTask)
	return
}
