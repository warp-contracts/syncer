package relay

import (
	"errors"
	"runtime"
	"strconv"
	"sync"
	"time"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/jackc/pgtype"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"

	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/warp-contracts/sequencer/app"
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

// Converts Arweave transactions into Warp's contracts
func NewParser(config *config.Config) (self *Parser) {
	self = new(Parser)

	// Decoding sequencer transactions
	proto.RegisterType((*sequencertypes.MsgDataItem)(nil), "sequencer.sequencer.MsgDataItem")
	proto.RegisterType((*sequencertypes.MsgArweaveBlock)(nil), "sequencer.sequencer.MsgArweaveBlock")

	encoding := app.MakeEncodingConfig()
	self.txConfig = encoding.TxConfig

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

func (self *Parser) parseMsgDataItem(msg cosmostypes.Msg) (interaction *model.Interaction, bundleItem *model.BundleItem, err error) {
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

	// FIXME: Parsing needs to be implemented, this is just a placeholder
	// Parse interaction from DataItem
	interaction, err = self.parser.Parse(&dataItem.DataItem, 0, arweave.Base64String{0}, time.Now().UnixMilli(), "", "")
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
		InteractionID: interaction.ID,
		DataItem:      pgtype.Bytea{Bytes: dataItemBytes, Status: pgtype.Present},
		Tags:          pgtype.JSONB{Bytes: []byte("{}"), Status: pgtype.Present},
	}

	tags := dataItem.DataItem.Tags.Append([]bundlr.Tag{
		{
			Name:  "Sequencer",
			Value: "RedStone",
		},
		// FIXME: Is this the right value for owner? What about ethereum?
		{
			Name:  "Sequencer-Owner",
			Value: dataItem.DataItem.Owner.Base64(),
		},
		{
			Name:  "Sequencer-Tx-Id",
			Value: interaction.InteractionId.Base64(),
		},
		{
			Name:  "Sequencer-Block-Height",
			Value: strconv.FormatInt(interaction.BlockHeight, 10),
		},
		{
			Name:  "Sequencer-Block-Id",
			Value: "",
		},
		{
			Name:  "Sequencer-Block-Timestamp",
			Value: "",
		},
		{
			Name:  "Sequencer-Mills",
			Value: dataItem.DataItem.Owner.Base64(),
		},
		{
			Name:  "Sequencer-Sort-Key",
			Value: dataItem.DataItem.Owner.Base64(),
		},
		{
			Name:  "Sequencer-Prev-Sort-Key",
			Value: dataItem.DataItem.Owner.Base64(),
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
func (self *Parser) parseTransaction(txBytes types.Tx) (interactions []*model.Interaction, bundleItems []*model.BundleItem, arweaveBlocks []*ArweaveBlock, err error) {
	// Decode transaction
	// TODO: Is this routine-safe?
	tx, err := self.txConfig.TxDecoder()(txBytes)
	if err != nil {
		return
	}

	// Parse interactions
	var (
		interaction  *model.Interaction
		bundleItem   *model.BundleItem
		arweaveBlock *sequencertypes.MsgArweaveBlock
	)

	for _, msg := range tx.GetMsgs() {
		switch proto.MessageName(msg) {
		case "sequencer.sequencer.MsgArweaveBlock":
			//  L1 interactions. Part of the data, rest will be downloaded from Arweave
			arweaveBlock, err = self.parseMsgArweaveBlock(msg)
			if err != nil {
				self.Log.WithError(err).Error("Failed to parse MsgArweaveBlock")
				return
			}
			arweaveBlocks = append(arweaveBlocks, &ArweaveBlock{Message: arweaveBlock})
		case "sequencer.sequencer.MsgDataItem":
			// L2 interactions, all needed data
			interaction, bundleItem, err = self.parseMsgDataItem(msg)
			if err != nil {
				self.Log.WithError(err).Error("Failed to parse MsgDataItem")
				return
			}
			interactions = append(interactions, interaction)
			bundleItems = append(bundleItems, bundleItem)
		default:
			self.Log.WithField("name", proto.MessageName(msg)).Warn("Unknown message type")
			continue
		}
	}

	return
}

func (self *Parser) parseBlock(block *types.Block) (out *Payload, err error) {
	// Meta info about the block
	out = new(Payload)
	out.SequencerBlockHash = string(block.DataHash)
	out.SequencerBlockHeight = block.Height
	out.SequencerBlockTimestamp = block.Time.UnixMilli()

	if len(block.Txs) == 0 {
		return
	}

	// Parse transactions in parallel
	var wg sync.WaitGroup
	wg.Add(len(block.Txs))
	var mtx sync.Mutex

	out.Interactions = make([]*model.Interaction, 0, len(block.Txs)+1000)
	out.BundleItems = make([]*model.BundleItem, 0, len(block.Txs)+1)

	for _, txBytes := range block.Txs {
		txBytes := txBytes

		self.SubmitToWorker(func() {
			interactions, bundleItems, arweaveBlocks, err := self.parseTransaction(txBytes)
			if err != nil {
				// FIXME: Monitoring
				self.Log.WithError(err).Error("Failed to parse interaction, skipping")
				goto done
			}

			mtx.Lock()
			out.Interactions = append(interactions, interactions...)
			out.BundleItems = append(bundleItems, bundleItems...)
			out.ArweaveBlocks = append(out.ArweaveBlocks, arweaveBlocks...)
			mtx.Unlock()

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
			self.Log.WithError(err).Error("Failed to parse block")
			panic(err)
		}

		self.Log.WithField("height", block.Height).Debug("Parsed block")

		select {
		case <-self.Ctx.Done():
			err = errors.New("task closing")
			return
		case self.Output <- payload:
		}
	}

	return nil
}
