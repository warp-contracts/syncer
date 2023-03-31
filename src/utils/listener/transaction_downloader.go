package listener

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/smartweave"
	"syncer/src/utils/task"
	"time"

	"github.com/cenkalti/backoff/v4"
)

// Fills in transactions for a given block
type TransactionDownloader struct {
	*task.Task

	client  *arweave.Client
	monitor monitoring.Monitor
	filter  func(*arweave.Transaction) bool
	input   chan *arweave.Block
	Output  chan *Payload

	// Parameters
	maxElapsedTime time.Duration
	maxInterval    time.Duration
}

// Using Arweave client periodically checks for blocks of transactions
func NewTransactionDownloader(config *config.Config) (self *TransactionDownloader) {
	self = new(TransactionDownloader)

	// No time limit by default
	self.filter = func(tx *arweave.Transaction) bool { return true }

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "transaction-downloader").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.ListenerNumWorkers, config.ListenerWorkerQueueSize).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *TransactionDownloader) WithMonitor(monitor monitoring.Monitor) *TransactionDownloader {
	self.monitor = monitor
	return self
}

func (self *TransactionDownloader) WithClient(client *arweave.Client) *TransactionDownloader {
	self.client = client
	return self
}

func (self *TransactionDownloader) WithInputChannel(v chan *arweave.Block) *TransactionDownloader {
	self.input = v
	return self
}

func (self *TransactionDownloader) WithBackoff(maxElapsedTime, maxInterval time.Duration) *TransactionDownloader {
	self.maxElapsedTime = maxElapsedTime
	self.maxInterval = maxInterval
	return self
}

func (self *TransactionDownloader) WithFilterContracts() *TransactionDownloader {
	self.filter = func(tx *arweave.Transaction) bool {
		if tx.Format < 2 {
			return false
		}

		var isContract bool
		var isContractSrcId bool

		for _, tag := range tx.Tags {
			if string(tag.Value) == "SmartWeaveContract" &&
				string(tag.Name) == smartweave.TagAppName {
				isContract = true
			}

			if string(tag.Name) == smartweave.TagContractSrcTxId && len(tag.Value) > 0 {
				isContractSrcId = true
			}

		}
		return isContract && isContractSrcId
	}
	return self
}

// Listens for changed height and downloads the missing blocks
func (self *TransactionDownloader) run() error {
	// Listen for new blocks (blocks)
	// Finishes when Listener is stopping
	for block := range self.input {
		// self.Log.
		// 	WithField("height", block.Height).
		// 	Debug("Downloading transactions")

		transactions, err := self.downloadTransactions(block)
		if self.IsStopping.Load() {
			// Neglect trhose transactions
			return nil
		}
		if err != nil {
			// FIXME: Monitor counter
			// FIXME: Shouldn't we stop?
			self.Log.WithError(err).WithField("height", block.Height).Error("Failed to download transactions in block")
			continue
		}

		self.monitor.GetReport().TransactionDownloader.State.TransactionsDownloaded.Add(uint64(len(transactions)))

		// Blocks until a monitorTranactions is ready to receive
		// or Listener is stopped
		self.Output <- &Payload{
			BlockHash:      block.IndepHash.Bytes(),
			BlockHeight:    block.Height,
			BlockTimestamp: block.Timestamp,
			Transactions:   transactions,
		}

	}

	return nil
}

func (self *TransactionDownloader) downloadTransactions(block *arweave.Block) (out []*arweave.Transaction, err error) {
	self.Log.Debug("Start downloading transactions...")
	defer self.Log.Debug("...Stopped downloading transactions")
	// Sync between workers
	var mtx sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(block.Txs))

	out = make([]*arweave.Transaction, 0, len(block.Txs))
	for _, txIdBytes := range block.Txs {
		txIdBytes := txIdBytes

		self.SubmitToWorker(func() {
			var (
				err error
				tx  *arweave.Transaction
			)

			// Encode txId for later
			txId := base64.RawURLEncoding.EncodeToString(txIdBytes)

			// Configure backoff
			b := backoff.NewExponentialBackOff()
			b.MaxElapsedTime = self.maxElapsedTime
			b.MaxInterval = self.maxInterval

			// Retries downloading transaction until success or permanent error
			err = backoff.Retry(
				func() (err error) {
					tx, err = self.client.GetTransactionById(self.Ctx, txId)
					if err != nil {
						if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
							// Stopping
							return backoff.Permanent(err)
						}
						self.Log.WithError(err).WithField("txId", txId).Warn("Failed to download transaction, retrying after timeout")

						// This will completly reset the HTTP client and possibly help in solving the problem
						self.client.Reset()

						self.monitor.GetReport().TransactionDownloader.Errors.TxDownloadErrors.Inc()

						// FIXME: Inform downstream something's wrong
					}

					return err
				}, b)
			if err != nil {
				// Permanent error
				self.Log.WithError(err).WithField("txId", txId).Error("Failed to download transaction, giving up")
				goto end
			}

			// Skip transactions that don't pass the filter
			if !self.filter(tx) {
				goto end
			}

			// Verify transaction signature
			err = tx.Verify()
			if err != nil {
				self.monitor.GetReport().Syncer.Errors.TxValidationErrors.Inc()
				self.Log.Error("Transaction failed to verify")
				goto end
			}

			// Add to output
			mtx.Lock()
			out = append(out, tx)
			mtx.Unlock()

			self.Log.WithField("txId", txId).Trace("Downloaded transaction")

		end:
			wg.Done()
		})
	}

	// Wait for workers to finish
	wg.Wait()

	return
}
