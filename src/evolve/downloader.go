package evolve

import (
	"context"
	"errors"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"

	"golang.org/x/exp/slices"
)

type Downloader struct {
	*task.Task

	client  *arweave.Client
	input   <- chan string
	Output  chan *model.ContractSource
}

func NewDownloader(config *config.Config) (self *Downloader) {
	self = new(Downloader)

	self.Output = make(chan *model.ContractSource)

	self.Task = task.NewTask(config, "evolve-downloader").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.Evolve.DownloaderNumWorkers, config.Evolve.DownloaderWorkerQueueSize).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Downloader) WithClient(client *arweave.Client) *Downloader {
	self.client = client
	return self
}

func (self *Downloader) WithInputChannel(v chan string) *Downloader {
	self.input = v
	return self
}

func (self *Downloader) run() (err error) {
	
	for srcId := range self.input {
		srcId := srcId
		var contractSrc *model.ContractSource
		self.SubmitToWorker(func() {
			tx, errSrc := self.downloadContractSrcTx(srcId)
			if errSrc != nil {
				self.Log.WithError(errSrc).
					WithField("srcId", srcId).
					Error("Failed to download contract source transaction")
				contractSrc.SrcTxId = srcId
				err = contractSrc.Src.Set("error")
				if err != nil {
					err = errors.New("could not set contract source error")
					return
				}
				goto end
			} 

			contractSrc, errSrc = self.loadContractSrcMetadata(tx)
			if errSrc != nil {
				self.Log.WithError(errSrc).
					WithField("srcId", srcId).
					Error("Failed to load contract source metadata")
				contractSrc.SrcTxId = srcId
				err = contractSrc.Src.Set("error")
				if err != nil {
					err = errors.New("could not set contract source error")
					return
				}
			}

			end:
				self.Output <- contractSrc
		})
	}

	return nil
}


func (self *Downloader) downloadContractSrcTx(srcId string) (out *arweave.Transaction, err error) {
	err = task.NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(0).
		WithMaxInterval(self.Config.Evolve.DownloaderSourceTransactiondMaxInterval).
		WithAcceptableDuration(self.Config.Evolve.DownloaderSourceTransactiondMaxInterval * 3).
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
			out, err = self.client.GetTransactionById(self.Ctx, srcId)
			if err != nil {
				self.Log.WithField("txId", srcId).Error("Failed to download contract source transaction")
				return err
			}
			return err
		})
	if err != nil {
		self.Log.WithError(err).WithField("txId", srcId).Error("Failed to download contract source transaction, giving up")
		return
	}
	
	return
}

func (self *Downloader) loadContractSrcMetadata(srcTx *arweave.Transaction) (out *model.ContractSource, err error) {
			out = model.NewContractSource()

			out.SrcTxId = srcTx.ID.Base64()

			srcContentType, ok := srcTx.GetTag(smartweave.TagContentType)
			if !ok {
				err = errors.New("contract source content type is not set")
				return
			}
		
			if !slices.Contains(self.Config.Contract.LoaderSupportedContentTypes, srcContentType) {
				err = errors.New("unsupported contract source content type")
				return
			}
		
			err = out.SrcContentType.Set(srcContentType)
			if err != nil {
				err = errors.New("could not set contract source type")
				return
			}

			owner, err := warp.GetWalletAddress(srcTx)
			if err != nil {
				self.Log.WithError(err).Error("could not get wallet address")
				return
			}
		
			err = out.Owner.Set(owner)
			if err != nil {
				err = errors.New("could not set owner")
				return
			}

			err = out.SrcTx.Set(srcTx)
			if err != nil {
				err = errors.New("could not set contract source transaction")
				return
			}

			err = srcTx.Verify()
			if err != nil {
				err = errors.New("could not verify contract source transaction")
				return
			}

			src, err := self.client.GetTransactionDataById(self.Ctx, srcTx)
			if err != nil {
				self.Log.WithError(err).Error("could not get contract source data")
				return
			}
		
			if out.IsJS() {
				err = out.Src.Set(src.String())
				if err != nil {
					err = errors.New("could not set contract source data")
					return
				}
			} else {
				srcWasmLang, ok := srcTx.GetTag(warp.TagWasmLang)
				if !ok {
					err = errors.New("WASM contract source language is not set")
					return
				}
				err = out.SrcWasmLang.Set(srcWasmLang)
				if err != nil {
					err = errors.New("could not set WASM contract source language")
					return
				}
			
				err = out.SrcBinary.Set(src.Bytes())
				if err != nil {
					err = errors.New("could not set contract source binary")
					return
				}
			}

		return
}