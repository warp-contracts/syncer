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
	"syncer/src/utils/warp"

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
			// FIXME: Handle error
			self.Log.WithError(err).Error("Failed to load contracts")
			return err
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
	self.Log.Debug("Start loading contracts...")
	defer self.Log.Debug("...Stopped loading contracts")

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

			b := backoff.NewExponentialBackOff()
			b.MaxElapsedTime = self.Config.Contract.LoaderBackoffMaxElapsedTime
			b.MaxInterval = self.Config.Contract.LoaderBackoffMaxInterval

			var contractData *ContractData

			// Retry loading contract upon error
			// Skip contract after LoaderBackoffMaxElapsedTime
			err := backoff.Retry(func() (err error) {
				contractData, err = self.load(tx)
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
				return
			}, b)
			if err != nil {
				// FIXME: Monitor errors
				self.Log.WithError(err).WithField("id", tx.ID).Error("Failed to load contract, stopped trying!")
				goto done
			}

			mtx.Lock()
			out = append(out, contractData)
			mtx.Unlock()

		done:
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

	_, ok := tx.GetTag(warp.TagWarpTestnet)
	if ok {
		// PPE: this does not make sens to me...we def. want to  sync the testnet contracts
		// - although now I'm aware that this is fucked-up in the current syncing code :-)
		err = errors.New("Trying to use testnet contract in a non-testnet env")
		return
	}

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

// PPE: remove?
//	let update: any = {
//		src_tx_id: definition.srcTxId,
//		init_state: definition.initState,
//		owner: definition.owner,
//		type,

//		pst_ticker: type == 'pst' ? definition.initState?.ticker : null,
//		pst_name: type == 'pst' ? definition.initState?.name : null,
//		contract_tx: { tags: definition.contractTx.tags },
//	 };

func (self *Loader) getContract(tx *arweave.Transaction) (out *model.Contract, err error) {
	self.Log.WithField("id", tx.ID).Debug("-> getContract")
	defer self.Log.WithField("id", tx.ID).Debug("<- getContract")

	var ok bool
	out = model.NewContract()

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
	owner, err := warp.GetAddress(tx)
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

		// PPE: pstInitState.Name ?
		err = out.PstName.Set(pstInitState.Ticker)
		if err != nil {
			return
		}
	}

	// PPE: where is the block_height and block_timestmap being set?
	// I don't see sync_timestamp being set (it was added recently)
	// Same with deployment_type (should be 'arweave')

	return
}

func (self *Loader) getSource(srcId string) (out *model.ContractSource, err error) {
	self.Log.WithField("src_tx_id", srcId).Debug("-> getSource")
	defer self.Log.WithField("src_tx_id", srcId).Debug("<- getSource")

	var ok bool
	out = model.NewContractSource()

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
	// --> getSource? getInitState rather :-)
	self.Log.WithField("id", contractTx.ID).Debug("--> getSource")
	defer self.Log.WithField("id", contractTx.ID).Debug("<-- getSource")

	self.Log.WithField("id", contractTx.ID).Debug("1")
	initState, ok := contractTx.GetTag(warp.TagInitState)
	if ok {
		// Init state in tags
		out.WriteString(initState)
		return
	}

	self.Log.WithField("id", contractTx.ID).Debug("2")
	initStateTxId, ok := contractTx.GetTag(warp.TagInitStateTx)
	if ok {
		// FIXME: Validate tags, eg. this value should be a valid transaction id
		// Init state in a separate transaction
		return self.client.GetTransactionDataById(self.Ctx, initStateTxId)
	}

	self.Log.WithField("id", contractTx.ID).Debug("3")
	// Init state is the contract's data
	if len(contractTx.Data) > 0 {
		// PPE will this produce a proper JSON (i.e. we're writing binary data here)?
		out.Write(contractTx.Data)
		return
	}

	self.Log.WithField("id", contractTx.ID).Debug("4")
	// It didn't fit into the data field, fetch chunks
	return self.client.GetChunks(self.Ctx, contractTx.ID)
}
