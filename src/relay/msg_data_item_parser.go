package relay

import (
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"runtime"
	"sync"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/jackc/pgtype"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"
	"golang.org/x/exp/slices"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
)

// Parses Sequencer's blocks into payload
type MsgDataItemParser struct {
	*task.Task

	monitor monitoring.Monitor

	parser *warp.DataItemParser

	input  <-chan *Payload
	Output chan *Payload
}

var (
	sortKeyRegExp = regexp.MustCompile(`^\d{12},\d{13},\d{8}$`)
)

// Converts Arweave transactions into Warp's contracts
func NewMsgDataItemParser(config *config.Config) (self *MsgDataItemParser) {
	self = new(MsgDataItemParser)

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

func (self *MsgDataItemParser) WithMonitor(monitor monitoring.Monitor) *MsgDataItemParser {
	self.monitor = monitor
	return self
}

func (self *MsgDataItemParser) WithInputChannel(v <-chan *Payload) *MsgDataItemParser {
	self.input = v
	return self
}

func (self *MsgDataItemParser) validateSortKey(interaction *model.Interaction, payload *Payload, idxInBlock int) (err error) {
	// Check previous sort key
	if interaction.LastSortKey.Status == pgtype.Present && !sortKeyRegExp.MatchString(interaction.LastSortKey.String) {
		err = errors.New("invalid prev sort key")
		return
	}

	// Check sort key
	if !sortKeyRegExp.MatchString(interaction.SortKey) {
		err = errors.New("invalid sort key")
		return
	}

	var arweaveHeight, sequencerHeight, idx int64
	_, err = fmt.Sscanf(interaction.SortKey, "%d,%d,%d", &arweaveHeight, &sequencerHeight, &idx)
	if err != nil {
		return
	}

	if sequencerHeight != payload.SequencerBlockHeight {
		err = errors.New("invalid sequencer height in sort key")
		self.Log.WithField("sort_key", interaction.SortKey).
			WithField("parsed_sequencer_height", sequencerHeight).
			WithField("sequencer_height", payload.SequencerBlockHeight).
			Error("Invalid sequencer height in sort key")
		return
	}

	if idx != int64(idxInBlock) {
		err = errors.New("invalid index in sort key")
		self.Log.WithField("sort_key", interaction.SortKey).
			WithField("parsed_index", idx).
			WithField("sequencer_index", idxInBlock).
			Error("Invalid index in sort key")
		return
	}

	if arweaveHeight != int64(payload.LastArweaveBlock.Height) {
		err = errors.New("invalid arweave height in sort key")
		self.Log.WithField("sort_key", interaction.SortKey).
			WithField("parsed_arweave_height", arweaveHeight).
			WithField("expected_arweave_height", payload.LastArweaveBlock.Height).
			Error("Invalid arweave height in sort key")
		return
	}

	return
}

func (self *MsgDataItemParser) validateSortKeys(payload *Payload, interactions []*model.Interaction) (err error) {
	// Downlaod order may differ from the order in the block
	sortedInteractions := append(make([]*model.Interaction, 0, len(interactions)), interactions...)
	slices.SortFunc(sortedInteractions, func(a *model.Interaction, b *model.Interaction) bool {
		return a.SortKey < b.SortKey
	})

	for idx, dataItem := range sortedInteractions {
		err = self.validateSortKey(dataItem, payload, idx)
		if err != nil {
			self.Log.WithError(err).Error("Failed to validate sort key")
			return
		}
	}
	return
}

func (self *MsgDataItemParser) parseMessage(msg cosmostypes.Msg, payload *Payload) (interaction *model.Interaction, bundleItem *model.BundleItem, err error) {
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
	arweaveBlockHash, err := arweave.Base64StringFromBase64(payload.LastArweaveBlock.Hash)
	if err != nil {
		self.Log.WithError(err).Error("Failed to decode block hash")
		return
	}

	interaction, err = self.parser.Parse(&dataItem.DataItem, int64(payload.LastArweaveBlock.Height), arweaveBlockHash, int64(payload.LastArweaveBlock.Timestamp), dataItem.SortKey, dataItem.PrevSortKey)
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

func (self *MsgDataItemParser) parse(payload *Payload) (err error) {
	// Parse transactions in parallel
	var wg sync.WaitGroup
	wg.Add(len(payload.Messages))
	var mtx sync.Mutex

	payload.Interactions = make([]*model.Interaction, 0, len(payload.Messages))
	payload.BundleItems = make([]*model.BundleItem, 0, len(payload.Messages))

	for i := range payload.Messages {
		i := i

		self.SubmitToWorker(func() {
			var (
				interaction *model.Interaction
				bundleItem  *model.BundleItem
				errMsg      error
			)

			if proto.MessageName(payload.Messages[i]) != "sequencer.sequencer.MsgDataItem" {
				goto done
			}

			interaction, bundleItem, errMsg = self.parseMessage(payload.Messages[i], payload)

			mtx.Lock()
			defer mtx.Unlock()

			if errMsg != nil {
				self.monitor.GetReport().Relayer.Errors.SequencerPermanentParsingError.Inc()
				self.Log.WithError(err).
					WithField("idx", i).
					WithField("sequencer_height", payload.SequencerBlockHeight).
					Error("Failed to parse MsgDataItem")
				goto done
			}

			// Append data
			payload.Interactions = append(payload.Interactions, interaction)
			payload.BundleItems = append(payload.BundleItems, bundleItem)

			// Update monitoring
			self.monitor.GetReport().Relayer.State.SequencerTransactionsParsed.Inc()

		done:
			wg.Done()
		})
	}

	// Wait for all transactions to be parsed
	wg.Wait()

	err = self.validateSortKeys(payload, payload.Interactions)
	if err != nil {
		self.Log.WithField("sequencer_height", payload.SequencerBlockHeight).WithError(err).Error("Failed to validate sort key")
		return
	}

	return
}

func (self *MsgDataItemParser) run() (err error) {
	// Each payload has a slice of transactions
	for payload := range self.input {
		err = self.parse(payload)
		if err != nil {
			if self.IsStopping.Load() {
				// Neglect those transactions, we're stopping anyway
				return nil
			}

			self.Log.WithField("sequencer_height", payload.SequencerBlockHeight).WithError(err).Error("Failed to parse block")

			// Stop everything
			// We can't neglect parsing errors
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
