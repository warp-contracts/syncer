package relay

import (
	"time"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
)

// Task that saves transactions to the payload
// It reuses listener.TransactionOrchestrator
type TransactionOrchestrator struct {
	*task.Task

	client  *arweave.Client
	monitor monitoring.Monitor

	input  chan *Payload
	Output chan *Payload

	transactionInput  chan *listener.Payload
	TransactionOutput chan *arweave.Block

	// Parameters
	maxElapsedTime time.Duration
	maxInterval    time.Duration

	lastBlockHeight int64
	lastBlockHash   arweave.Base64String
}

// Using Arweave client periodically checks for blocks of transactions
func NewTransactionOrchestrator(config *config.Config) (self *TransactionOrchestrator) {
	self = new(TransactionOrchestrator)

	self.Output = make(chan *Payload)
	self.TransactionOutput = make(chan *arweave.Block)

	self.Task = task.NewTask(config, "transaction-orchestrator").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
			close(self.TransactionOutput)
		})

	return
}

func (self *TransactionOrchestrator) WithInputChannel(v chan *Payload) *TransactionOrchestrator {
	self.input = v
	return self
}

func (self *TransactionOrchestrator) WithTransactionInput(v chan *listener.Payload) *TransactionOrchestrator {
	self.transactionInput = v
	return self
}

// Listens for changed height and downloads the missing blocks
func (self *TransactionOrchestrator) run() (err error) {
	for payload := range self.input {
		// Download transactions one by one using TransactionDownloader
		for i, arweaveBlock := range payload.ArweaveBlocks {
			// Send out block for processing
			select {
			case <-self.Ctx.Done():
				return nil
			case self.TransactionOutput <- arweaveBlock.Block:
			}

			// Receive downloaded transactions
			select {
			case <-self.Ctx.Done():
				return nil
			case p, ok := <-self.transactionInput:
				if !ok {
					self.Log.Error("Transaction input channel closed")
					return nil
				}
				payload.ArweaveBlocks[i].Transactions = p.Transactions
			}
			self.Log.WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
				Info("Downloaded transactions from one arweave block")
		}

		// Arweave blocks filled
		self.Output <- payload
	}

	return nil
}
