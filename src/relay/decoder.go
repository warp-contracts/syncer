package relay

import (
	"errors"
	"runtime"
	"sync"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	proto "github.com/cosmos/gogoproto/proto"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"

	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
)

// Parses Sequencer's blocks into payload
// This is the first step - decoding sequencer's transactions into messages
// Messages are processed in another task
type Decoder struct {
	*task.Task

	monitor monitoring.Monitor

	txConfig client.TxConfig

	input  chan *types.Block
	Output chan *Payload
}

// Converts Arweave transactions into Warp's contracts
func NewDecoder(config *config.Config) (self *Decoder) {
	self = new(Decoder)

	// Decoding sequencer transactions
	registry := codectypes.NewInterfaceRegistry()
	sequencertypes.RegisterInterfaces(registry)
	marshaler := codec.NewProtoCodec(registry)
	self.txConfig = tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "decoder").
		WithSubtaskFunc(self.run).
		WithWorkerPool(runtime.NumCPU(), 1).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Decoder) WithMonitor(monitor monitoring.Monitor) *Decoder {
	self.monitor = monitor
	return self
}

func (self *Decoder) WithInputChannel(v chan *types.Block) *Decoder {
	self.input = v
	return self
}

// Used by other tasks
func (self *Decoder) Decode(txBytes []byte) (tx cosmostypes.Tx, err error) {
	tx, err = self.txConfig.TxDecoder()(txBytes)
	return
}

// Transaction consists of multiple messages
func (self *Decoder) decodeTx(txBytes []byte, block *types.Block) (msg cosmostypes.Msg, err error) {
	// Decode transaction
	tx, err := self.Decode(txBytes)
	if err != nil {
		return
	}

	// Only one msg in tx is allowed
	if len(tx.GetMsgs()) != 1 {
		err = errors.New("transaction must have exactly one message")
		return
	}

	msg = tx.GetMsgs()[0]

	// Whitelist of messages
	switch proto.MessageName(msg) {
	case "sequencer.sequencer.MsgArweaveBlock":
	case "sequencer.sequencer.MsgDataItem":
	default:
		self.Log.WithField("name", proto.MessageName(msg)).Error("Unknown message type")
		err = errors.New("unknown sequencer message type")
	}

	return
}

func (self *Decoder) decodeBlock(block *types.Block) (out *Payload, err error) {
	// Meta info about the block
	out = new(Payload)
	out.SequencerBlockHash = block.DataHash
	out.SequencerBlockHeight = block.Height
	out.SequencerBlockTimestamp = block.Time.UnixMilli()

	if len(block.Txs) == 0 {
		return
	}

	// Parse transactions in parallel
	var wg sync.WaitGroup
	wg.Add(len(block.Txs))
	var mtx sync.Mutex

	out.Messages = make([]cosmostypes.Msg, len(block.Txs))

	for i := range block.Txs {
		i := i

		self.SubmitToWorker(func() {
			message, errTx := self.decodeTx(block.Txs[i], block)

			mtx.Lock()
			defer mtx.Unlock()

			if errTx != nil {
				self.monitor.GetReport().Relayer.Errors.SequencerPermanentParsingError.Inc()
				self.Log.WithError(errTx).
					WithField("idx", i).
					WithField("sequencer_height", block.Height).
					Error("Failed to decode transaction from sequencer")

				err = errTx
				goto done
			}

			out.Messages[i] = message

			// Update monitoring
			self.monitor.GetReport().Relayer.State.SequencerTransactionsParsed.Inc()
		done:
			wg.Done()
		})
	}

	// Wait for all transactions to be decoded
	wg.Wait()

	return
}

func (self *Decoder) run() (err error) {
	// Each payload has a slice of transactions
	var payload *Payload
	for block := range self.input {
		payload, err = self.decodeBlock(block)
		if err != nil {
			if self.IsStopping.Load() {
				// Neglect those transactions, we're stopping anyway
				return nil
			}

			self.Log.WithField("sequencer_height", block.Height).WithError(err).Error("Failed to decode block, stopping")

			// Stop everything
			// We can't neglect parsing errors
			panic(err)
		}

		select {
		case <-self.Ctx.Done():
			err = errors.New("task closing")
			return
		case self.Output <- payload:
		}
	}

	return nil
}
