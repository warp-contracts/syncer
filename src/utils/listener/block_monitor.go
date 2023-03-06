package listener

import (
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitor"
	"syncer/src/utils/task"
	"time"

	"gorm.io/gorm"
)

// Task that periodically checks for new arweave network info.
// Optionally waits for a number of required confirmation blocks before emitting the info
type BlockMonitor struct {
	*task.Task

	// Runtime configuration
	startHeight int64

	client  *arweave.Client
	monitor *monitor.Monitor

	input  chan *arweave.NetworkInfo
	Output chan *Payload
}

// Using Arweave client periodically checks for blocks of transactions
func NewBlockMonitor(config *config.Config) (self *BlockMonitor) {
	self = new(BlockMonitor)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "block-monitor").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.ListenerNumWorkers).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *BlockMonitor) WithMonitor(monitor *monitor.Monitor) *BlockMonitor {
	self.monitor = monitor
	return self
}

func (self *BlockMonitor) WithClient(client *arweave.Client) *BlockMonitor {
	self.client = client
	return self
}

func (self *BlockMonitor) WithInitStartHeight(db *gorm.DB) *BlockMonitor {
	self.Task = self.Task.WithOnBeforeStart(func() (err error) {
		// Get the last storeserverd block height from the database
		var state model.State
		err = db.WithContext(self.Ctx).First(&state).Error
		if err != nil {
			self.Log.WithError(err).Error("Failed to get last transaction block height")
			return
		}
		self.startHeight = state.LastTransactionBlockHeight
		return nil
	})
	return self
}

func (self *BlockMonitor) WithInputChannel(v chan *arweave.NetworkInfo) *BlockMonitor {
	self.input = v
	return self
}

// Listens for changed height and downloads the missing blocks
func (self *BlockMonitor) run() error {
	lastSyncedHeight := self.startHeight

	// Listen for new blocks (blocks)
	// Finishes when Listener is stopping
	for networkInfo := range self.input {

		self.Log.
			WithField("last", lastSyncedHeight).
			WithField("new", networkInfo.Height).
			WithField("numNewBlocks", networkInfo.Height-lastSyncedHeight).
			Debug("Discovered new blocks")

		// Download transactions from
		for height := lastSyncedHeight + 1; height <= networkInfo.Height; height++ {
			self.monitor.Report.SyncerCurrentHeight.Store(height)

		retry:
			self.Log.WithField("height", height).Debug("Downloading block")

			block, err := self.client.GetBlockByHeight(self.Ctx, height)
			if err != nil {
				if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
					return nil
				}

				self.Log.WithError(err).WithField("height", height).Error("Failed to download block, retrying...")

				// This will completly reset the HTTP client and possibly help in solving the problem
				self.client.Reset()

				self.monitor.Report.Errors.BlockDownloadErrors.Inc()

				time.Sleep(self.Config.ListenerRetryFailedTransactionDownloadInterval)
				if self.IsStopping.Load() {
					// Neglect this block and close the goroutine
					return nil
				}

				goto retry
			}

			if !block.IsValid() {
				self.Log.WithField("height", height).Error("Block hash isn't valid")
				// self.Log.WithField("height", height).Error("Block hash isn't valid, blacklisting peer for ever and retrying")
				self.monitor.Report.Errors.BlockValidationErrors.Inc()
				time.Sleep(time.Second * 10)
				goto retry
			}

			self.Log.
				WithField("height", height).
				WithField("length", len(block.Txs)).
				Debug("Downloaded block")

			transactions, err := self.downloadTransactions(block)
			if self.IsStopping.Load() {
				// Neglect trhose transactions
				return nil
			}
			if err != nil {
				self.Log.WithError(err).WithField("height", height).Error("Failed to download transactions in block")
				continue
			}

			self.monitor.Report.TransactionsDownloaded.Add(uint64(len(transactions)))

			payload := &Payload{
				BlockHeight:  block.Height,
				Transactions: transactions,
			}

			// Blocks until a monitorTranactions is ready to receive
			// or Listener is stopped
			self.Output <- payload
		}
	}

	return nil
}

func (self *BlockMonitor) downloadTransactions(block *arweave.Block) (out []*arweave.Transaction, err error) {
	// Sync between workers
	var mtx sync.Mutex
	var wg sync.WaitGroup
	wg.Add(len(block.Txs))

	out = make([]*arweave.Transaction, len(block.Txs))
	for idx, txId := range block.Txs {
		idx := idx
		txId := base64.RawURLEncoding.EncodeToString(txId)

		self.Workers.Submit(func() {
			// NOTE: Infinite loop, because there's nothing better we can do.
			for {
				if self.IsStopping.Load() {
					goto end
				}

				self.Log.WithField("txId", txId).Debug("Downloading transaction")

				tx, err := self.client.GetTransactionById(self.Ctx, txId)
				if err != nil {
					if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
						goto end
					}
					self.Log.WithError(err).WithField("txId", txId).Error("Failed to download transaction, retrying after timeout")

					// This will completly reset the HTTP client and possibly help in solving the problem
					self.client.Reset()

					self.monitor.Report.Errors.TxDownloadErrors.Inc()

					time.Sleep(self.Config.ListenerRetryFailedTransactionDownloadInterval)
					if self.IsStopping.Load() {
						// Neglect this block and close the goroutine
						self.Log.WithError(err).WithField("txId", txId).Error("Neglect downloading transaction, listener is stopping anyway")
						goto end
					}

					continue
					// FIXME: Inform downstream something's wrong
				}

				mtx.Lock()
				out[idx] = tx
				mtx.Unlock()

				goto end
			}
		end:
			wg.Done()
		})
	}

	// Wait for workers to finish
	wg.Wait()

	return
}
