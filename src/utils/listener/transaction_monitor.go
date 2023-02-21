package listener

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitor"
	"syncer/src/utils/task"
	"syncer/src/utils/warp"
	"time"
)

// Task that periodically checks for new arweave network info.
// Optionally waits for a number of required confirmation blocks before emitting the info
type TransactionMonitor struct {
	*task.Task

	interactionParser *warp.InteractionParser
	monitor           *monitor.Monitor

	input  chan *Payload
	Output chan *Payload
}

// Using Arweave client periodically checks for blocks of transactions
func NewTransactionMonitor(config *config.Config, interval time.Duration) (self *TransactionMonitor) {
	self = new(TransactionMonitor)

	self.Task = task.NewTask(config, "transaction-monitor").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.ListenerNumWorkers).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	// Converting Arweave transactions to interactions
	var err error
	self.interactionParser, err = warp.NewInteractionParser(config)
	if err != nil {
		self.Log.Panic("Failed to initialize parser")
	}

	self.Output = make(chan *Payload)

	return
}

func (self *TransactionMonitor) WithMonitor(monitor *monitor.Monitor) *TransactionMonitor {
	self.monitor = monitor
	return self
}

func (self *TransactionMonitor) WithInput(v chan *Payload) *TransactionMonitor {
	self.input = v
	return self
}

// Listens for changed height and downloads the missing blocks
func (self *TransactionMonitor) run() error {
	var err error

	// Listen for downloaded transactions
	// Finishes when Listener is stopping, after self.input is closed
	for payload := range self.input {
		// Filter out transactions that do not have the matching tags
		payload.Transactions = self.filterTransactions(payload.Transactions)

		// Check signatures of all transactions
		err = self.verifyTransactions(payload.Transactions)
		if err != nil {
			continue
		}

		// Parse transactions into interactions
		payload.Interactions = make([]*model.Interaction, len(payload.Transactions))
		for i, tx := range payload.Transactions {
			payload.Interactions[i], err = self.interactionParser.Parse(tx, payload.BlockHeight, payload.BlockId, payload.BlockTimestamp)
			if err != nil {
				self.Log.WithField("tx_id", tx.ID).Warn("Failed to parse transaction")
				continue
			}
		}

		// Blocks until a upstream is ready to receive
		self.Output <- payload
	}

	return nil
}

func (self *TransactionMonitor) filterTransactions(transactions []*arweave.Transaction) (out []*arweave.Transaction) {
	out = make([]*arweave.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		for _, tag := range tx.Tags {
			// Format needst to be 2 in order for the verification to work
			if tx.Format == 2 &&
				string(tag.Value) == "SmartWeaveAction" &&
				string(tag.Name) == "App-Name" {
				out = append(out, tx)
				break
			}
		}
	}
	return
}

func (self *TransactionMonitor) verifyTransactions(transactions []*arweave.Transaction) (err error) {
	for _, tx := range transactions {
		err = tx.Verify()
		if err != nil {
			self.monitor.Increment(monitor.Kind(monitor.TxValidationErrors))
			self.Log.Error("Transaction failed to verify")
			return
		}
	}
	return
}
