package contract

import (
	"bytes"
	"errors"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
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
	input  chan *arweave.Transaction
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

func (self *Loader) WithInputChannel(v chan *arweave.Transaction) *Loader {
	self.input = v
	return self
}

func (self *Loader) WithClient(client *arweave.Client) *Loader {
	self.client = client
	return self
}

func (self *Loader) run() error {
	// Each payload has a slice of transactions
	for tx := range self.input {
		tx := tx
		self.SubmitToWorker(func() {
			err := self.load(tx)
			if err != nil {
				self.Log.WithError(err).WithField("id", tx.ID).Error("Failed to load contract")
			}
		})
	}

	return nil
}

func (self *Loader) load(tx *arweave.Transaction) (err error) {
	var payload Payload

	_, ok := tx.GetTag(warp.TagWarpTestnet)
	if ok {
		err = errors.New("Trying to use testnet contract in a non-testnet env")
		return
	}

	payload.Contract, err = self.getContract(tx)
	if err != nil {
		self.Log.WithError(err).Error("Failed to parse contract")
		return
	}

	// FIXME: Validate manifest
	// manifest, ok := tx.GetTag(warp.TagManifest)
	// if !ok {
	// 	return
	// }

	payload.Source, err = self.getSource(payload.Contract.SrcTxId.String)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get contract source")
		return
	}

	select {
	case <-self.Ctx.Done():
	case self.Output <- &payload:
	}

	return
}

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
	var ok bool
	out = new(model.Contract)

	// Source tx id
	srcTxId, ok := tx.GetTag(smartweave.TagContractSrcTxId)
	if !ok {
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
		self.Log.WithError(err).Error("Failed to get contract init state")
		return
	}
	err = out.InitState.Set(initStateBuffer.Bytes())
	if err != nil {
		return
	}

	// Try parsing init state as a PST
	pstInitState, err := warp.ParsePstInitState(initStateBuffer.Bytes())
	if err != nil {
		self.Log.WithError(err).Error("Failed to parse init state as JSON")
		return
	}
	if !pstInitState.IsPst() {
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

		err = out.PstName.Set(pstInitState.Ticker)
		if err != nil {
			return
		}
	}

	return
}

func (self *Loader) getSource(srcId string) (out *model.ContractSource, err error) {
	var ok bool
	out = new(model.ContractSource)

	srcTx, err := self.client.GetTransactionById(self.Ctx, srcId)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get contract source")
		return
	}

	// Verify tags
	out.SrcContentType, ok = srcTx.GetTag(smartweave.TagContentType)
	if !ok {
		err = errors.New("contract source content type is not set")
		return
	}

	if !slices.Contains(self.Config.Contract.LoaderSupportedContentTypes, out.SrcContentType) {
		err = errors.New("unsupported contract source content type")
		return
	}

	// Check signature
	err = srcTx.Verify()
	if err != nil {
		return
	}

	// Get source
	src, err := self.client.GetTransactionDataById(self.Ctx, srcId)
	if err != nil {
		self.Log.WithError(err).Error("Failed to get contract source")
		return
	}

	if out.IsJS() {
		out.Src = src.String()
	} else {
		out.SrcWasmLang, ok = srcTx.GetTag(warp.TagWasmLang)
		if !ok {
			err = errors.New("WASM contract language is not set")
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
	return self.client.GetChunks(self.Ctx, initStateTxId)
}
