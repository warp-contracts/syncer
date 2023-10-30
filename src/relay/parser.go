package relay

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"sync"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/jackc/pgtype"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"

	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
)

// Parses Sequencer's blocks into payload
type Parser struct {
	*task.Task

	monitor monitoring.Monitor

	txConfig client.TxConfig
	parser   *warp.DataItemParser

	input  chan *types.Block
	Output chan *Payload
}

var (
	sortKeyRegExp = regexp.MustCompile(`^\d{12},\d{13},\d{8}$`)
)

// Converts Arweave transactions into Warp's contracts
func NewParser(config *config.Config) (self *Parser) {
	self = new(Parser)

	// Decoding sequencer transactions
	registry := codectypes.NewInterfaceRegistry()
	sequencertypes.RegisterInterfaces(registry)
	marshaler := codec.NewProtoCodec(registry)
	self.txConfig = tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	// Parsing interactions
	self.parser = warp.NewDataItemParser(config)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "parser").
		WithSubtaskFunc(self.run).
		WithWorkerPool(runtime.NumCPU(), 1).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Parser) WithMonitor(monitor monitoring.Monitor) *Parser {
	self.monitor = monitor
	return self
}

func (self *Parser) WithInputChannel(v chan *types.Block) *Parser {
	self.input = v
	return self
}

func (self *Parser) validateSortKey(msg *sequencertypes.MsgDataItem, block *types.Block, idxInBlock int) (err error) {
	// Check previous sort key
	if msg.PrevSortKey != "" && !sortKeyRegExp.MatchString(msg.PrevSortKey) {
		err = errors.New("invalid prev sort key")
		return
	}

	// Check sort key
	if !sortKeyRegExp.MatchString(msg.SortKey) {
		err = errors.New("invalid sort key")
		return
	}

	var arweaveHeight, sequencerHeight, idx int
	_, err = fmt.Sscanf(msg.SortKey, "%d,%d,%d", &arweaveHeight, &sequencerHeight, &idx)
	if err != nil {
		return
	}

	if sequencerHeight != int(block.Height) {
		err = errors.New("invalid sequencer height in sort key")
		return
	}

	if idx != idxInBlock {
		err = errors.New("invalid sequencer height in sort key")
		return
	}

	// TODO: Validate arweave height

	return
}

func (self *Parser) parseMsgDataItem(msg cosmostypes.Msg, block *types.Block, idx int) (interaction *model.Interaction, bundleItem *model.BundleItem, err error) {
	var isDataItem bool
	dataItem, isDataItem := msg.(*sequencertypes.MsgDataItem)
	if !isDataItem {
		err = errors.New("failed to cast MsgDataItem")
		return
	}

	// Verify data
	err = dataItem.DataItem.Verify()
	if err != nil {
		self.Log.WithError(err).Error("Failed to verify data item")
		return
	}

	err = dataItem.DataItem.VerifySignature()
	if err != nil {
		self.Log.WithError(err).Error("Failed to verify data item signature")
		return
	}

	err = self.validateSortKey(dataItem, block, idx)
	if err != nil {
		self.Log.WithError(err).Error("Failed to validate sort key")
		return
	}

	// FIXME: Parsing needs to be implemented, this is just a placeholder
	// Parse interaction from DataItem
	interaction, err = self.parser.Parse(&dataItem.DataItem, 0, arweave.Base64String{0}, block.Time.UnixMilli(), dataItem.SortKey, dataItem.PrevSortKey)
	if err != nil {
		self.Log.WithError(err).Error("Failed to parse interaction")
		return
	}

	// Serialize data item
	dataItemBytes, err := dataItem.DataItem.Marshal()
	if err != nil {
		self.Log.WithError(err).Error("Failed to serialize data item")
		return
	}

	bundleItem = &model.BundleItem{
		InteractionID:  interaction.ID,
		Transaction:    pgtype.JSONB{Status: pgtype.Null},
		DataItem:       pgtype.Bytea{Bytes: dataItemBytes, Status: pgtype.Present},
		Tags:           pgtype.JSONB{Bytes: []byte("{}"), Status: pgtype.Present},
		BundlrResponse: pgtype.JSONB{Status: pgtype.Null},
		BlockHeight:    sql.NullInt64{Valid: false},
		State:          model.BundleStatePending,
	}

	tags := dataItem.DataItem.Tags.Append([]bundlr.Tag{
		{
			Name:  "Sequencer",
			Value: "RedStone",
		},
		// FIXME: Make sure this is correct for arweave and ethereum
		{
			Name:  "Sequencer-Owner",
			Value: dataItem.DataItem.Owner.Base64(),
		},
		{
			Name:  "Sequencer-Tx-Id",
			Value: interaction.InteractionId.Base64(),
		},
		{
			Name:  "Sequencer-Sort-Key",
			Value: dataItem.SortKey,
		},
		{
			Name:  "Sequencer-Prev-Sort-Key",
			Value: dataItem.PrevSortKey,
		},
		{
			Name:  "Sequencer-Random",
			Value: base64.RawURLEncoding.EncodeToString(dataItem.Random),
		},
	})

	err = bundleItem.Tags.Set(tags)
	if err != nil {
		return
	}

	return
}

func (self *Parser) parseMsgArweaveBlock(msg cosmostypes.Msg) (arweaveBlock *sequencertypes.MsgArweaveBlock, err error) {
	var isArweaveBlock bool
	arweaveBlock, isArweaveBlock = msg.(*sequencertypes.MsgArweaveBlock)
	if !isArweaveBlock {
		err = errors.New("failed to cast MsgArweaveBlock")
		return
	}

	return
}

// Transaction consists of multiple messages
func (self *Parser) parseTransaction(block *types.Block, idx int) (interaction *model.Interaction, bundleItem *model.BundleItem, arweaveBlock *ArweaveBlock, err error) {

	// Decode transaction
	tx, err := self.txConfig.TxDecoder()(block.Txs[idx])
	if err != nil {
		return
	}

	// Only one msg in tx
	if len(tx.GetMsgs()) != 1 {
		err = errors.New("transaction must have exactly one message")
		return
	}

	// Parse message
	msg := tx.GetMsgs()[0]

	switch proto.MessageName(msg) {
	case "sequencer.sequencer.MsgArweaveBlock":
		//  L1 interactions. Part of the data, rest will be downloaded from Arweave
		arweaveBlock = new(ArweaveBlock)
		arweaveBlock.Message, err = self.parseMsgArweaveBlock(msg)
		if err != nil {
			self.Log.WithError(err).Error("Failed to parse MsgArweaveBlock")
			return
		}

	case "sequencer.sequencer.MsgDataItem":
		// L2 interactions, all needed data
		if interaction != nil && bundleItem != nil {
			err = errors.New("two data items in one transaction")
			self.Log.WithError(err).Error("There can't be two data items in one transaction, signing wouldn't work")
			return
		}

		interaction, bundleItem, err = self.parseMsgDataItem(msg, block, idx)
		if err != nil {
			self.Log.WithError(err).Error("Failed to parse MsgDataItem")
			return
		}
	default:
		self.Log.WithField("name", proto.MessageName(msg)).Warn("Unknown message type")
		err = errors.New("unknown sequencer message type")
	}

	return
}

func (self *Parser) parseBlock(block *types.Block) (out *Payload, err error) {
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

	out.Interactions = make([]*model.Interaction, 0, len(block.Txs))
	out.BundleItems = make([]*model.BundleItem, 0, len(block.Txs))

	for i := range block.Txs {
		i := i

		self.SubmitToWorker(func() {
			interaction, bundleItem, arweaveBlock, err := self.parseTransaction(block, i)
			if err != nil {
				self.monitor.GetReport().Relayer.Errors.SequencerPermanentParsingError.Inc()
				self.Log.WithError(err).Error("Failed to parse transaction from sequencer, skipping")
				goto done
			}

			mtx.Lock()
			if interaction != nil && bundleItem != nil {
				// L2 interaction
				out.Interactions = append(out.Interactions, interaction)
				out.BundleItems = append(out.BundleItems, bundleItem)
			}
			if arweaveBlock != nil {
				out.ArweaveBlocks = append(out.ArweaveBlocks, arweaveBlock)
			}
			mtx.Unlock()

			// Update monitoring
			self.monitor.GetReport().Relayer.State.SequencerTransactionsParsed.Inc()
		done:
			wg.Done()
		})
	}

	wg.Wait()

	return
}

func (self *Parser) run() (err error) {
	// Each payload has a slice of transactions
	var payload *Payload
	for block := range self.input {
		payload, err = self.parseBlock(block)
		if err != nil {
			self.Log.WithField("sequencer_height", block.Height).WithError(err).Error("Failed to parse block")
			panic(err)
		}

		// self.Log.WithField("height", block.Height).Trace("Parsed block")

		select {
		case <-self.Ctx.Done():
			err = errors.New("task closing")
			return
		case self.Output <- payload:
		}
	}

	return nil
}
