package poller

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"time"

	"github.com/sirupsen/logrus"
)

type Poller struct {
	client *arweave.Client
	config *config.Config

	log *logrus.Entry

	// Stopping
	isStopping    *atomic.Bool
	stopChannel   chan bool
	stopOnce      *sync.Once
	stopWaitGroup sync.WaitGroup
	Ctx           context.Context
	cancel        context.CancelFunc

	// Internal state
	heightChannel       chan int64
	TransactionsChannel chan []*arweave.Transaction

	// Runtime configuration
	startHeight int64
}

// Using Arweave client periodically checks for blocks of transactions
// When a new block is found it downloads the transactions and emits them into a channel
func NewPoller(config *config.Config) (self *Poller) {
	self = new(Poller)
	self.config = config

	// Stopping
	self.stopOnce = &sync.Once{}
	self.isStopping = &atomic.Bool{}
	self.stopWaitGroup = sync.WaitGroup{}
	self.stopChannel = make(chan bool)

	self.client = arweave.NewClient(config)
	self.log = logger.NewSublogger("poller")

	// Context active as long as there's anything running in Poller
	self.Ctx, self.cancel = context.WithCancel(context.Background())

	// A chain of channels
	self.heightChannel = make(chan int64)
	self.TransactionsChannel = make(chan []*arweave.Transaction)
	return
}

func (self *Poller) WithStartHeight(v int64) *Poller {
	self.startHeight = v
	return self
}

func (self *Poller) run(f func()) {
	self.stopWaitGroup.Add(1)
	go func() {
		defer func() {
			// run() finished, so it's time to cancel Poller's context
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
				self.log.WithError(err).Error("Panic in Poller. Stopping.")

				panic(p)
			}
		}()
		f()
	}()
}

func (self *Poller) Start() {
	self.run(self.monitorNetwork)
	self.run(self.monitorBlocks)
}

// Periodically checks Arweave network info for updated height
func (self *Poller) monitorNetwork() {
	ticker := time.NewTicker(self.config.StoreMaxTimeInQueue)

	var lastHeight int64
	for {
		select {
		case <-self.stopChannel:
			close(self.heightChannel)
			return
		case <-ticker.C:
			networkInfo, err := self.client.GetNetworkInfo(self.Ctx)
			if err != nil {
				self.log.Error("Failed to get Arweave network info")
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
func (self *Poller) monitorBlocks() {
	lastSyncedHeight := self.startHeight

	// Listen for new blocks (blocks)
	// Finishes when Poller is stopping
	for presentHeight := range self.heightChannel {

		// Download transactions from
		for height := lastSyncedHeight + 1; height <= presentHeight; height++ {
			block, err := self.client.GetBlockByHeight(self.Ctx, height)
			if err != nil {
				self.log.WithField("height", height).Error("Failed to download block")
				continue
				// FIXME: Inform downstream something's wrong
			}

			transactions := make([]*arweave.Transaction, len(block.Txs))
			for idx, txId := range block.Txs {
				tx, err := self.client.GetTransactionById(self.Ctx, txId)
				if err != nil {
					self.log.WithField("height", height).Error("Failed to download block")
					// FIXME: Inform downstream something's wrong
				}
				transactions[idx] = tx
			}

			self.TransactionsChannel <- transactions
		}
	}

	// Monitoring Network stopped, so it's safe to close
	// We don't wait for stopChannel because we want to process pending network infos before stopping
	close(self.TransactionsChannel)
}

func (self *Poller) StopWait() {
	self.stopOnce.Do(func() {
		// Signals that we're stopping
		close(self.stopChannel)

		// Mark that we're stopping
		self.isStopping.Store(true)
	})

	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, failed to finish poller")
	case <-self.Ctx.Done():
		self.log.Info("Listener finished")
	}
}
