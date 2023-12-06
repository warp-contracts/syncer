package listener

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
	"github.com/warp-contracts/syncer/src/utils/task"

	"github.com/cenkalti/backoff/v4"
)

// Fills in transactions for a given block
type TransactionDownloader struct {
	*task.Task

	client             *arweave.Client
	monitor            monitoring.Monitor
	filter             func(*arweave.Transaction) bool
	getTransactionData func(*arweave.Transaction) bool
	input              chan *arweave.Block
	Output             chan *Payload

	// Parameters
	maxElapsedTime time.Duration
	maxInterval    time.Duration
}

// Using Arweave client periodically checks for blocks of transactions
func NewTransactionDownloader(config *config.Config) (self *TransactionDownloader) {
	self = new(TransactionDownloader)

	// No time limit by default
	self.filter = func(tx *arweave.Transaction) bool { return true }

	self.getTransactionData = func(tx *arweave.Transaction) bool { return false }

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "transaction-downloader").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.TransactionDownloader.NumWorkers, config.TransactionDownloader.WorkerQueueSize).
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
		if tx == nil || tx.Format < 2 {
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

func (self *TransactionDownloader) WithFilterInteractions() *TransactionDownloader {
	self.filter = func(tx *arweave.Transaction) bool {
		isInteraction, err := smartweave.ValidateInteraction(tx)
		if err != nil {
			self.Log.WithField("txId", tx.ID.Base64()).WithError(err).Warn("neglecting invalid interaction")
			return false
		}
		return isInteraction
	}
	self.getTransactionData = smartweave.InteractionWithData
	return self
}

// Listens for changed height and downloads the missing blocks
func (self *TransactionDownloader) run() error {
	// Listen for new blocks (blocks)
	// Finishes when Listener is stopping
	for block := range self.input {
		transactions, err := self.downloadTransactions(block)
		if self.IsStopping.Load() {
			// Neglect trhose transactions
			return nil
		}
		if err != nil {
			// FIXME: Shouldn't we stop?
			self.Log.WithError(err).WithField("height", block.Height).Error("Failed to download transactions in block")
			continue
		}

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
	if len(block.Txs) == 0 {
		//Skip
		return
	}
	self.Log.WithField("height", block.Height).WithField("len", len(block.Txs)).Debug("Start downloading transactions...")
	defer self.Log.WithField("height", block.Height).Debug("...Stopped downloading transactions")
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

			// Retries downloading transaction until success or permanent error
			err = task.NewRetry().
				WithContext(self.Ctx).
				WithMaxElapsedTime(self.maxElapsedTime).
				WithMaxInterval(self.maxInterval).
				WithAcceptableDuration(self.maxInterval * 2).
				WithOnError(func(err error, isDurationAcceptable bool) error {
					self.Log.WithError(err).WithField("txId", txId).Warn("Failed to download transaction, retrying after timeout")

					if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
						// Stopping
						return backoff.Permanent(err)
					}
					self.monitor.GetReport().TransactionDownloader.Errors.Download.Inc()

					if errors.Is(err, arweave.ErrPending) {
						// https://docs.arweave.org/developers/server/http-api#undefined-4
						// This is a temporary error, after some time the transaction will be available
						time.Sleep(time.Second)
						return err
					}

					if errors.Is(err, arweave.ErrOverspend) {
						// This is a permanent error
						return backoff.Permanent(err)
					}

					if !isDurationAcceptable {
						// This will completly reset the HTTP client and possibly help in solving the problem
						self.client.Reset()
					}

					return err
				}).
				Run(func() error {
					tx, err = self.client.GetTransactionById(self.Ctx, txId)
					if err != nil {
						self.Log.WithField("tx", txId).Error("Failed to download transaction")
						return err
					}

					// Skip transactions in unsupported format
					if tx.Format < 2 {
						tx = nil
						return nil
					}

					// Verify transaction signature.
					// Peer might be malicious and send us invalid transaction for this id
					err = tx.Verify()
					if err != nil {
						self.Log.WithField("tx", txId).Error("Transaction failed to verify, retry downloading...")
						self.monitor.GetReport().TransactionDownloader.Errors.Validation.Inc()
					}

					if self.getTransactionData(tx) {
						tx.Data, err = self.downloadTransactionData(tx)
						if err != nil {
							self.monitor.GetReport().TransactionDownloader.Errors.DataDownload.Inc()
							self.Log.WithField("tx", txId).Error("Failed to download transaction data, retry downloading...")
						}
					}

					return err
				})

			if err != nil {
				// Permanent error
				self.monitor.GetReport().TransactionDownloader.Errors.PermanentDownloadFailure.Inc()
				self.Log.WithError(err).WithField("txId", txId).Error("Failed to download proper transaction, giving up")
				goto end
			}

			// Update metrics
			self.monitor.GetReport().TransactionDownloader.State.TransactionsDownloaded.Inc()

			// Skip transactions that don't pass the filter
			if !self.filter(tx) {
				goto end
			}

			// Add to output
			mtx.Lock()
			out = append(out, tx)
			mtx.Unlock()

		end:
			wg.Done()
		})
	}

	// Wait for workers to finish
	wg.Wait()

	return
}

func (self *TransactionDownloader) downloadTransactionData(tx *arweave.Transaction) (arweave.Base64String, error) {
	buf, err := self.client.GetTransactionDataById(self.Ctx, tx)
	if err != nil {
		return nil, err
	}
	return arweave.Base64String(buf.Bytes()), nil
}
