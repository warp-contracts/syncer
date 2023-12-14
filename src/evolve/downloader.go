package evolve

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
)

type Downloader struct {
	*task.Task

	client  *arweave.Client
	input   <-chan string
	Output  chan *model.ContractSource
	monitor monitoring.Monitor
}

func NewDownloader(config *config.Config) (self *Downloader) {
	self = new(Downloader)

	self.Output = make(chan *model.ContractSource)

	self.Task = task.NewTask(config, "downloader").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.Evolver.DownloaderNumWorkers, config.Evolver.DownloaderWorkerQueueSize).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Downloader) WithClient(client *arweave.Client) *Downloader {
	self.client = client
	return self
}

func (self *Downloader) WithMonitor(monitor monitoring.Monitor) *Downloader {
	self.monitor = monitor
	return self
}

func (self *Downloader) WithInputChannel(v chan string) *Downloader {
	self.input = v
	return self
}

func (self *Downloader) run() (err error) {

	for srcId := range self.input {
		srcId := srcId
		self.SubmitToWorker(func() {

			contractSrc, errSrc := self.download(srcId)
			if errSrc != nil {
				if self.IsStopping.Load() {
					// Task is stopping, no further actions needed
					return
				}

				self.Log.WithError(errSrc).
					WithField("srcId", srcId).
					Error("Failed to download contract source, setting contract source as error")

				contractSrc.SrcTxId = srcId
				err = contractSrc.Src.Set("error")
				if err != nil {
					err = errors.New("could not set contract source error")
					return
				}
			}

			self.Output <- contractSrc
		})
	}

	return nil
}

func (self *Downloader) download(srcId string) (out *model.ContractSource, err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.Evolver.DownloaderSourceTransactiondMaxInterval).
		WithAcceptableDuration(self.Config.Evolver.DownloaderSourceTransactiondMaxInterval * 3).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			// Permanent errors
			if (errors.Is(err, context.Canceled) && self.IsStopping.Load()) || errors.Is(err, arweave.ErrNotFound) ||
				errors.Is(err, arweave.ErrOverspend) {
				return backoff.Permanent(err)
			}

			// Errors fo retry
			self.Log.WithError(err).WithField("srcId", srcId).Warn("Failed to download contract source transaction, retrying after timeout")

			if errors.Is(err, arweave.ErrPending) {
				time.Sleep(time.Second)
				return err
			}

			if !isDurationAcceptable {
				self.client.Reset()
			}

			return err
		}).
		Run(func() error {
			out, err = self.getContractSrc(srcId)
			if err != nil {
				self.Log.WithError(err).WithField("txId", srcId).Error("Failed to download contract source transaction")
				return err
			}
			return err
		})
	if err != nil {
		self.Log.WithError(err).WithField("txId", srcId).Error("Failed to download contract source transaction, giving up")
		self.monitor.GetReport().Evolver.Errors.DownloaderDownlaodError.Inc()

		return
	}

	self.monitor.GetReport().Evolver.State.DownloaderSourcesLoaded.Inc()
	return
}

func (self *Downloader) getContractSrc(srcId string) (out *model.ContractSource, err error) {
	out = model.NewContractSource()

	srcTx, err := self.client.GetTransactionById(self.Ctx, srcId)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get source transaction")
		return
	}

	if int(srcTx.DataSize.Int64()) > warp.MAX_TRANSACTION_DATA_SIZE {
		err = errors.New("Data size exceeds maximum transaction data size")
		return
	}

	err = warp.SetContractSourceMetadata(srcTx, out)
	if err != nil {
		self.Log.WithError(err).Error("Failed to set contract source metadata")
		return
	}

	src, err := self.client.GetTransactionDataById(self.Ctx, srcTx)
	if err != nil {
		self.Log.WithError(err).Error("could not get contract source data")
		return
	}

	err = warp.SetContractSource(src, srcTx, out)
	if err != nil {
		self.Log.WithError(err).Error("Failed to set contract source data")
		return
	}

	return
}
