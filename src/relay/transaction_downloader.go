package relay

import (
	"context"
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

	client  *arweave.Client
	monitor monitoring.Monitor
	input   chan *Payload
	Output  chan *Payload
}

// Using Arweave client periodically checks for blocks of transactions
func NewTransactionDownloader(config *config.Config) (self *TransactionDownloader) {
	self = new(TransactionDownloader)

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

func (self *TransactionDownloader) WithInputChannel(v chan *Payload) *TransactionDownloader {
	self.input = v
	return self
}

// Downloads one transaction, handles automatic retries and validation
func (self *TransactionDownloader) downloadOne(txId string) (out *arweave.Transaction, err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(0 /* never give up */).
		WithMaxInterval(self.Config.Relayer.ArweaveBlockDownloadMaxInterval).
		WithAcceptableDuration(self.Config.Relayer.ArweaveBlockDownloadMaxInterval * 3).
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
			out, err = self.client.GetTransactionById(self.Ctx, txId)
			if err != nil {
				self.Log.WithField("txId", txId).Error("Failed to download transaction")
				return err
			}

			if smartweave.IsInteractionWithData(out) {
				buf, err := self.client.GetTransactionDataById(self.Ctx, out)
				if err != nil {
					self.Log.WithField("txId", txId).Error("Failed to download transaction data")
					self.monitor.GetReport().TransactionDownloader.Errors.DataDownload.Inc()
					return err
				}
				out.Data = arweave.Base64String(buf.Bytes())
			}

			// Check if transaction is a valid interaction
			isInteraction, err := smartweave.ValidateInteraction(out)
			if !isInteraction {
				err = errors.New("tx is not an interaction")
			}
			if err != nil {
				self.Log.WithField("txId", txId).WithError(err).Error("Transaction is not a valid interaction")
				return err
			}

			// Verify transaction signature.
			// Peer might be malicious and send us invalid transaction for this id
			err = out.Verify()
			if err != nil {
				self.Log.WithField("txId", txId).Error("Transaction failed to verify, retry downloading...")
				self.monitor.GetReport().TransactionDownloader.Errors.Validation.Inc()
				return err
			}

			return err
		})
	if err != nil {
		// Permanent error
		self.monitor.GetReport().TransactionDownloader.Errors.PermanentDownloadFailure.Inc()
		self.Log.WithError(err).WithField("txId", txId).Error("Failed to download proper transaction, giving up")
		return
	}

	// Update metrics
	self.monitor.GetReport().TransactionDownloader.State.TransactionsDownloaded.Inc()

	return
}

func (self *TransactionDownloader) downloadTransactions(block *ArweaveBlock) (out []*arweave.Transaction, err error) {
	if len(block.Message.Transactions) == 0 {
		// Skip, we'll only update info about the arweave block
		return
	}

	self.Log.WithField("arweave_height", block.Block.Height).WithField("len", len(block.Message.Transactions)).Debug("Start downloading transactions...")
	defer self.Log.WithField("arweave_height", block.Block.Height).Debug("...Stopped downloading transactions")

	// Sync between workers
	var mtx sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(block.Message.Transactions))

	out = make([]*arweave.Transaction, len(block.Message.Transactions))
	for i, txInfo := range block.Message.Transactions {
		txInfo := txInfo
		i := i

		self.SubmitToWorker(func() {
			tx, errOne := self.downloadOne(txInfo.Transaction.Id)
			mtx.Lock()
			if errOne != nil {
				err = errOne
			} else {
				out[i] = tx
			}
			mtx.Unlock()
			wg.Done()
		})
	}

	// Wait for workers to finish
	wg.Wait()

	return
}

// Fills in transactions for arweave blocks in payload
func (self *TransactionDownloader) run() (err error) {
	for payload := range self.input {
		// Download transactions one by one using TransactionDownloader
		for i, arweaveBlock := range payload.ArweaveBlocks {
			transactions, err := self.downloadTransactions(arweaveBlock)
			if err != nil {
				if self.IsStopping.Load() {
					// Neglect those transactions, we're stopping anyway
					return nil
				}
				self.Log.WithError(err).
					WithField("sequencer_height", payload.SequencerBlockHeight).
					WithField("arweave_height", arweaveBlock.Block.Height).
					Error("Failed to one of the transactions")

				// Stop everything
				// We can't neglect missing transactions
				panic(err)
			}
			payload.ArweaveBlocks[i].Transactions = transactions
			self.Log.WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
				Info("Downloaded transactions from one arweave block")
		}

		// Arweave blocks filled
		select {
		case <-self.Ctx.Done():
			return nil
		case self.Output <- payload:
		}

	}

	return nil
}
