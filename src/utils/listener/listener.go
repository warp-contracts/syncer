package listener

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"syncer/src/utils/arweave"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syncer/src/utils/model"
	"syncer/src/utils/warp"
	"time"

	"github.com/sirupsen/logrus"
)

type Listener struct {
	client            *arweave.Client
	config            *config.Config
	log               *logrus.Entry
	interactionParser *warp.InteractionParser

	// Stopping
	isStopping    *atomic.Bool
	stopChannel   chan bool
	stopOnce      *sync.Once
	stopWaitGroup sync.WaitGroup
	Ctx           context.Context
	cancel        context.CancelFunc

	// Internal state
	heightChannel       chan int64
	TransactionsChannel chan *Payload
	PayloadChannel      chan *Payload

	// Runtime configuration
	startHeight int64
}

// Using Arweave client periodically checks for blocks of transactions
// When a new block is found it downloads the transactions and emits them into a channel
//
// +-----------+               +-------------+                     +----------------+
// |           |               |             |                     |                |
// |  Network  | heightChannel |   Block     | TransactionChannel  |   Transaction  |  PayloadChannel
// |  Monitor  +--------------->   Monitor   +--------------------->   Monitor      +------------------>...
// |           |               |             |                     |                |
// +-----------+               +-------------+                     +----------------+
func NewListener(config *config.Config) (self *Listener) {
	self = new(Listener)
	self.log = logger.NewSublogger("listener")
	self.config = config

	// Listener context, active as long as there's anything running in Listener
	self.Ctx, self.cancel = context.WithCancel(context.Background())
	self.Ctx = common.SetConfig(self.Ctx, config)

	// Stopping
	self.stopOnce = &sync.Once{}
	self.isStopping = &atomic.Bool{}
	self.stopWaitGroup = sync.WaitGroup{}
	self.stopChannel = make(chan bool, 1)

	// A chain of channels
	self.heightChannel = make(chan int64)
	self.TransactionsChannel = make(chan *Payload, config.ListenerQueueSize)
	self.PayloadChannel = make(chan *Payload, config.ListenerQueueSize)

	// Converting Arweave transactions to interactions
	var err error
	self.interactionParser, err = warp.NewInteractionParser(config)
	if err != nil {
		self.log.Panic("Failed to initialize parser")
	}
	return
}

func (self *Listener) WithClient(client *arweave.Client) *Listener {
	self.client = client
	return self
}

func (self *Listener) WithStartHeight(v int64) *Listener {
	self.startHeight = v
	return self
}

func (self *Listener) run(f func()) {
	self.stopWaitGroup.Add(1)
	go func() {
		defer func() {
			// run() finished, so it's time to cancel Listener's context
			// NOTE: This is the only place self.Ctx is cancelled
			// self.cancel()
			self.stopWaitGroup.Done()

			var err error
			if p := recover(); p != nil {
				switch p := p.(type) {
				case error:
					err = p
				default:
					err = fmt.Errorf("%s", p)
				}
				self.log.WithError(err).Error("Panic in Listener. Stopping.")

				panic(p)
			}
		}()
		f()
	}()
}

func (self *Listener) Start() {
	self.run(self.monitorNetwork)
	self.run(self.monitorBlocks)
	self.run(self.monitorTransactions)
}

// Periodically checks Arweave network info for updated height
func (self *Listener) monitorNetwork() {
	ticker := time.NewTicker(self.config.StoreMaxTimeInQueue)

	var lastHeight int64
	for {
		select {
		case <-self.stopChannel:
			self.log.Debug("Closing heightChannel")
			close(self.heightChannel)
			return
		case <-ticker.C:
			networkInfo, err := self.client.GetNetworkInfo(self.Ctx)
			if err != nil {
				self.log.WithError(err).Error("Failed to get Arweave network info")
				continue
			}

			if networkInfo.Height <= lastHeight {
				continue
			}

			// There are new blocks, broadcast
			lastHeight = networkInfo.Height

			// Writing is a blocking operation
			// There needs to be a goroutine ready (@see monitorBlocks) to download the blocks
			// This can't timeout, but can be stopped
			select {
			case <-self.stopChannel:
			case self.heightChannel <- lastHeight:
			}
		}
	}
}

// Listens for changed height and downloads the missing blocks
func (self *Listener) monitorBlocks() {
	lastSyncedHeight := self.startHeight

	// Listen for new blocks (blocks)
	// Finishes when Listener is stopping
	for presentHeight := range self.heightChannel {

		self.log.
			WithField("last", lastSyncedHeight).
			WithField("new", presentHeight).
			WithField("numNewBlocks", presentHeight-lastSyncedHeight).
			Debug("Discovered new blocks")

		// Download transactions from
		for height := lastSyncedHeight + 1; height <= presentHeight; height++ {
			self.log.WithField("height", height).Debug("Downloading block")

			block, err := self.client.GetBlockByHeight(self.Ctx, height)
			if err != nil {
				self.log.WithError(err).WithField("height", height).Error("Failed to download block")
				continue
				// FIXME: Inform downstream something's wrong
			}

			self.log.
				WithField("height", height).
				WithField("length", len(block.Txs)).
				Debug("Downloaded block")

			transactions := make([]*arweave.Transaction, len(block.Txs))
			for idx, txId := range block.Txs {
			retry:
				self.log.WithField("txId", txId).Trace("Downloading transaction")

				tx, err := self.client.GetTransactionById(self.Ctx, txId)
				if err != nil {
					self.log.WithError(err).WithField("txId", txId).Error("Failed to download transaction, retrying after timeout")

					time.Sleep(2 * time.Second)
					if self.isStopping.Load() {
						// Neglect this block and close the goroutine
						goto end
					}

					goto retry
					// FIXME: Inform downstream something's wrong
				}
				transactions[idx] = tx
			}

			payload := &Payload{
				BlockHeight:  block.Height,
				Transactions: transactions,
			}

			// Blocks until a monitorTranactions is ready to receive
			// or Listener is stopped
			self.TransactionsChannel <- payload
		}
	}

end:
	// Monitoring Network stopped, so it's safe to close
	// We don't wait for stopChannel because we want to process pending network infos before stopping
	self.log.Debug("Closing TransactionsChannel")
	close(self.TransactionsChannel)
}

// Listens for downloaded transactions and processes them
func (self *Listener) monitorTransactions() {

	var err error

	// Listen for downloaded transactions
	// Finishes when Listener is stopping, after self.TransactionsChannel is closed
	for payload := range self.TransactionsChannel {
		// Filter out transactions that do not have the matching tags
		payload.Transactions = self.filterTransactions(payload.Transactions)

		// Parse transactions into interactions
		payload.Interactions = make([]*model.Interaction, len(payload.Transactions))
		for i, tx := range payload.Transactions {
			payload.Interactions[i], err = self.interactionParser.Parse(tx, payload.BlockHeight, payload.BlockId, payload.BlockTimestamp)
			if err != nil {
				self.log.WithField("tx_id", tx.ID).Warn("Failed to parse transaction")
				continue
			}
		}

		// Blocks until a upstream is ready to receive
		self.PayloadChannel <- payload
	}

	// Downloading blocks/transactions stopped, so it's safe to close
	// We don't wait for stopChannel
	// because we want to process pending data before stopping
	self.log.Debug("Closing PayloadChannel")
	close(self.PayloadChannel)
}

func (self *Listener) Stop() {
	self.log.Info("Stopping Listener...")
	self.stopOnce.Do(func() {
		// Signals that we're stopping
		close(self.stopChannel)

		// Mark that we're stopping
		self.isStopping.Store(true)
	})
}

func (self *Listener) filterTransactions(transactions []*arweave.Transaction) (out []*arweave.Transaction) {
	out = make([]*arweave.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		for _, tag := range tx.Tags {
			if tag.Value == "SmartWeaveAction" && tag.Name == "App-Name" {
				out = append(out, tx)
				break
			}
		}
	}
	return
}

func (self *Listener) StopWait() {
	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	self.Stop()

	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, failed to finish Listener")
	case <-self.Ctx.Done():
		self.log.Info("Listener finished")
	}
}
