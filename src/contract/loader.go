package contract

import (
	"bytes"
	"errors"
	"sync"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/smartweave"
	"syncer/src/utils/task"
	"syncer/src/utils/tool"
	"syncer/src/utils/warp"
	"time"

	"github.com/cenkalti/backoff/v4"
	"golang.org/x/exp/slices"
)

// Gets contract's source and init state
type Loader struct {
	*task.Task
	monitor monitoring.Monitor
	client  *arweave.Client
	// Data about the interactions that need to be bundled
	input  chan *listener.Payload
	Output chan *Payload
}

// Converts Arweave transactions into Warp's contracts
func NewLoader(config *config.Config) (self *Loader) {
	self = new(Loader)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "contract-loader").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.ListenerNumWorkers, config.ListenerWorkerQueueSize).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Loader) WithMonitor(monitor monitoring.Monitor) *Loader {
	self.monitor = monitor
	return self
}

func (self *Loader) WithInputChannel(v chan *listener.Payload) *Loader {
	self.input = v
	return self
}

func (self *Loader) WithClient(client *arweave.Client) *Loader {
	self.client = client
	return self
}

func (self *Loader) run() error {
	// Each payload has a slice of transactions
	for payload := range self.input {
		data, err := self.loadAll(payload.Transactions)
		if err != nil {
			// This should never happen
			self.Log.WithError(err).WithField("height", payload.BlockHeight).Error("Failed to load all contracts from a block")
			return err
		}

		// Fill in common fields
		for i := range data {
			data[i].Contract.BlockHeight = uint64(payload.BlockHeight)
			data[i].Contract.BlockTimestamp = uint64(payload.BlockTimestamp)
		}

		select {
		case <-self.Ctx.Done():
			return nil
		case self.Output <- &Payload{
			Data:        data,
			BlockHeight: uint64(payload.BlockHeight),
			BlockHash:   payload.BlockHash,
		}:
		}
	}

	return nil
}

func (self *Loader) loadAll(transactions []*arweave.Transaction) (out []*ContractData, err error) {
	if len(transactions) > 0 {
		// Skip
		return
	}

	self.Log.WithField("len", len(transactions)).Debug("Start loading contracts...")
	defer self.Log.WithField("len", len(out)).Debug("...Stopped loading contracts")

	var (
		wg  sync.WaitGroup
		mtx sync.Mutex
	)

	// Wait for all the contracts to be processed
	wg.Add(len(transactions))

	out = make([]*ContractData, 0, len(transactions))
	for _, tx := range transactions {
		tx := tx
		self.SubmitToWorker(func() {
			self.Log.WithField("id", tx.ID).Debug("Worker loading contract...")
			defer self.Log.WithField("id", tx.ID).Debug("...Worker loading contract")

			// Retry loading contract upon error
			// Skip contract after LoaderBackoffMaxElapsedTime
			err := task.NewRetry().
				WithMaxElapsedTime(self.Config.Contract.LoaderBackoffMaxElapsedTime).
				WithMaxInterval(self.Config.Contract.LoaderBackoffMaxInterval).
				WithOnError(func(err error) {
					// FIXME: Monitor errors
					self.Log.WithError(err).WithField("id", tx.ID).Debug("Retrying loading contract")
				}).
				Run(func() (err error) {
					contractData, err := self.load(tx)
					if err != nil {
						if err == arweave.ErrNotFound {
							// FIXME: Monitor errors
							// No need to retry if any of the data is not found
							// Arweave client already retries with multiple peers
							self.Log.WithError(err).WithField("id", tx.ID).Error("Failed to load contract, couldn't download source or init state")
							return backoff.Permanent(err)
						}
						// FIXME: Monitor errors
						self.Log.WithError(err).WithField("id", tx.ID).Warn("Failed to load contract, retrying after timeout...")
					}

					mtx.Lock()
					out = append(out, contractData)
					mtx.Unlock()
					return
				})
			if err != nil {
				// FIXME: Monitor errors
				self.Log.WithError(err).WithField("id", tx.ID).Error("Failed to load contract, stopped trying!")
			}

			wg.Done()
		})
	}

	// Wait for all contracts in batch to be loaded
	wg.Wait()
	return
}

func (self *Loader) load(tx *arweave.Transaction) (out *ContractData, err error) {
	self.Log.WithField("id", tx.ID).Debug("Start loading contract...")
	defer self.Log.WithField("id", tx.ID).Debug("...Stop loading contract")

	out = new(ContractData)

	out.Contract, err = self.getContract(tx)
	if err != nil {
		self.Log.WithError(err).WithField("id", tx.ID).Error("Failed to parse contract")
		return
	}

	// FIXME: Validate manifest
	// manifest, ok := tx.GetTag(warp.TagManifest)
	// if !ok {
	// 	return
	// }

	out.Source, err = self.getSource(out.Contract.SrcTxId.String)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get contract source")
		return
	}

	return
}

func (self *Loader) getContract(tx *arweave.Transaction) (out *model.Contract, err error) {
	self.Log.WithField("id", tx.ID).Debug("-> getContract")
	defer self.Log.WithField("id", tx.ID).Debug("<- getContract")

	var ok bool
	out = model.NewContract()
	out.ContractId = tx.ID
	err = out.DeploymentType.Set("arweave")
	if err != nil {
		return
	}

	err = out.SyncTimestamp.Set(time.Now().Unix())
	if err != nil {
		return
	}

	// Source tx id
	srcTxId, ok := tx.GetTag(smartweave.TagContractSrcTxId)
	if !ok {
		err = errors.New("missing contract source tx id")
		return
	}
	err = out.SrcTxId.Set(srcTxId)
	if err != nil {
		return
	}

	// Owner
	owner, err := warp.GetWalletAddress(tx)
	if err != nil {
		return
	}
	err = out.Owner.Set(owner)
	if err != nil {
		return
	}

	// Contract tx
	err = out.ContractTx.Set(tx)
	if err != nil {
		return
	}

	// Init state
	initStateBuffer, err := self.getInitState(tx)
	if err != nil {
		self.Log.WithError(err).WithField("id", tx.ID).Error("Failed to get contract init state")
		return
	}

	if !tool.IsJSON(initStateBuffer.Bytes()) {
		err = backoff.Permanent(errors.New("init state is not valid JSON"))
		return
	}

	err = out.InitState.Set(initStateBuffer.Bytes())
	if err != nil {
		return
	}

	// Try parsing init state as a PST
	pstInitState, err := warp.ParsePstInitState(initStateBuffer.Bytes())
	if err != nil || !pstInitState.IsPst() {
		err = out.Type.Set(model.ContractTypeOther)
		if err != nil {
			return
		}
	} else {
		err = out.Type.Set(model.ContractTypePst)
		if err != nil {
			return
		}

		err = out.PstTicker.Set(pstInitState.Ticker)
		if err != nil {
			return
		}

		err = out.PstName.Set(pstInitState.Name)
		if err != nil {
			return
		}
	}

	return
}

func (self *Loader) getSource(srcId string) (out *model.ContractSource, err error) {
	self.Log.WithField("src_tx_id", srcId).Debug("-> getSource")
	defer self.Log.WithField("src_tx_id", srcId).Debug("<- getSource")

	var ok bool
	out = model.NewContractSource()
	out.SrcTxId = srcId

	srcTx, err := self.client.GetTransactionById(self.Ctx, srcId)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get contract source transaction")
		return
	}

	// Verify tags
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
		return
	}

	// Check signature
	err = srcTx.Verify()
	if err != nil {
		return
	}

	// Get source from transaction's data
	src, err := self.client.GetTransactionDataById(self.Ctx, srcId)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get source data")
		return
	}

	if out.IsJS() {
		err = out.Src.Set(src.String())
		if err != nil {
			return
		}
	} else {
		srcWasmLang, ok := srcTx.GetTag(warp.TagWasmLang)
		if !ok {
			err = errors.New("WASM contract language is not set")
			return
		}
		err = out.SrcWasmLang.Set(srcWasmLang)
		if err != nil {
			return
		}

		err = out.SrcBinary.Set(src.Bytes())
		if err != nil {
			return
		}
	}
	return
}

func (self *Loader) getInitState(contractTx *arweave.Transaction) (out bytes.Buffer, err error) {
	self.Log.WithField("id", contractTx.ID).Debug("--> getInitState")
	defer self.Log.WithField("id", contractTx.ID).Debug("<-- getInitState")

	initState, ok := contractTx.GetTag(warp.TagInitState)
	if ok {
		// Init state in tags
		out.WriteString(initState)
		return
	}

	initStateTxId, ok := contractTx.GetTag(warp.TagInitStateTx)
	if ok {
		// FIXME: Validate tags, eg. this value should be a valid transaction id
		// Init state in a separate transaction
		return self.client.GetTransactionDataById(self.Ctx, initStateTxId)
	}

	// Init state is the contract's data
	if len(contractTx.Data) > 0 {
		out.Write(contractTx.Data)
		return
	}

	// It didn't fit into the data field, fetch chunks
	return self.client.GetChunks(self.Ctx, contractTx.ID)
}
