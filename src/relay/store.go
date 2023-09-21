package relay

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/jackc/pgtype"
	"github.com/warp-contracts/sequencer/app"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles saving data to the database in na robust way.
// - groups incoming Interactions into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Store struct {
	*task.Processor[*types.Block, *types.Block]

	DB      *gorm.DB
	monitor monitoring.Monitor

	savedBlockHeight  uint64
	finishedTimestamp uint64
	finishedHeight    uint64
	finishedBlockHash []byte
	txConfig          client.TxConfig
	parser            *warp.DataItemParser
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	// Decoding sequencer transactions
	proto.RegisterType((*sequencertypes.MsgDataItem)(nil), "sequencer.sequencer.MsgDataItem")
	encoding := app.MakeEncodingConfig()
	self.txConfig = encoding.TxConfig

	// Parsing interactions
	self.parser = warp.NewDataItemParser(config)

	self.Processor = task.NewProcessor[*types.Block, *types.Block](config, "store").
		WithBatchSize(config.Relayer.StoreBatchSize).
		WithOnFlush(config.Relayer.StoreMaxTimeInQueue, self.flush).
		WithOnProcess(self.process).
		WithBackoff(0, config.Relayer.StoreMaxBackoffInterval)

	return
}

func (self *Store) WithMonitor(v monitoring.Monitor) *Store {
	self.monitor = v
	return self
}

func (self *Store) WithInputChannel(v chan *types.Block) *Store {
	self.Processor = self.Processor.WithInputChannel(v)
	return self
}

func (self *Store) WithDB(v *gorm.DB) *Store {
	self.DB = v
	return self
}

func (self *Store) process(block *types.Block) (out []*types.Block, err error) {
	self.Log.WithField("height", block.Height).Debug("Processing")
	self.finishedTimestamp = uint64(block.Time.UnixMilli())
	self.finishedHeight = uint64(block.Height)
	self.finishedBlockHash = block.LastCommitHash
	out = []*types.Block{block}
	return
}

func (self *Store) getDataItem(tx cosmostypes.Tx) (out *sequencertypes.MsgDataItem, err error) {
	for _, msg := range tx.GetMsgs() {
		if proto.MessageName(msg) != "sequencer.sequencer.MsgDataItem" {
			continue
		}

		var isDataItem bool
		out, isDataItem = msg.(*sequencertypes.MsgDataItem)
		if !isDataItem {
			err = errors.New("failed to cast")
			return
		}

		return
	}

	err = errors.New("no data item")
	return
}

func (self *Store) parseMsgDataItem(msg cosmostypes.Msg) (interaction *model.Interaction, bundleItem *model.BundleItem, err error) {
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

	bundleItem.Tags.Set(tags)
	return
}

// Transaction consists of multiple messages
func (self *Store) parseTransaction(txBytes types.Tx) (interactions []*model.Interaction, dataItems []*model.BundleItem, err error) {
	// Decode transaction
	tx, err := self.txConfig.TxDecoder()(txBytes)
	if err != nil {
		return
	}

	// Parse interactions
	for _, msg := range tx.GetMsgs() {
		switch proto.MessageName(msg) {
		case "sequencer.sequencer.MsgDataItem":
			interaction, dataItem, err := self.parseMsgDataItem(msg)
			if err != nil {
				self.Log.WithError(err).Error("Failed to parse MsgDataItem")
				return nil, nil, err
			}
			interactions = append(interactions, interaction)
			dataItems = append(dataItems, dataItem)
		}

	}

	return
}

func (self *Store) parseBlock(block *types.Block) (interactions []*model.Interaction, bundleItems []*model.BundleItem, err error) {
	if len(block.Txs) == 0 {
		return
	}
	var wg sync.WaitGroup
	wg.Add(len(block.Txs))
	var mtx sync.Mutex

	interactions = make([]*model.Interaction, 0, len(block.Txs))
	bundleItems = make([]*model.BundleItem, 0, len(block.Txs))

	for _, txBytes := range block.Txs {
		txBytes := txBytes

		self.SubmitToWorker(func() {
			interactionsFromTx, bundleItemsFromTx, err := self.parseTransaction(txBytes)
			if err != nil {
				// FIXME: Monitoring
				self.Log.WithError(err).Error("Failed to parse interaction, skipping")
				goto done
			}

			mtx.Lock()
			interactions = append(interactions, interactionsFromTx...)
			bundleItems = append(bundleItems, bundleItemsFromTx...)
			mtx.Unlock()

		done:
			wg.Done()
		})
	}

	wg.Wait()

	return
}

func (self *Store) flush(blocks []*types.Block) (out []*types.Block, err error) {
	if self.savedBlockHeight == self.finishedHeight && len(blocks) == 0 {
		// No need to flush, nothing changed
		return
	}

	// Interactions from all blocks
	var (
		interactions, interactionsFromBlock []*model.Interaction
		bundleItems, bundleItemsFromBlock   []*model.BundleItem
	)

	// Parse interactions block by block
	for _, block := range blocks {
		interactionsFromBlock, bundleItemsFromBlock, err = self.parseBlock(block)
		if err != nil {
			self.Log.WithError(err).Error("Failed to parse interaction, skipping")
			continue
		}
		interactions = append(interactions, interactionsFromBlock...)
		bundleItems = append(bundleItems, bundleItemsFromBlock...)

	}

	// Set sync timestamp
	now := time.Now().UnixMilli()
	for _, interaction := range interactions {
		err = interaction.SyncTimestamp.Set(now)
		if err != nil {
			self.Log.WithError(err).Error("Failed to set sync_timestamp")
			return
		}
	}

	err = self.DB.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) error {
			if self.finishedHeight <= 0 {
				return errors.New("block height too small")
			}
			err = tx.WithContext(self.Ctx).
				Model(&model.State{
					Name: model.SyncedComponentRelayer,
				}).
				Updates(model.State{
					FinishedBlockTimestamp: self.finishedTimestamp,
					FinishedBlockHeight:    self.finishedHeight,
					FinishedBlockHash:      self.finishedBlockHash,
				}).
				Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to update last transaction block height")
				return err
			}

			// No interactions and bundle items to save
			if len(interactions) == 0 {
				return nil
			}

			// Save interactions
			err = tx.WithContext(self.Ctx).
				Table("interactions").
				Clauses(clause.OnConflict{
					DoNothing: true,
					Columns:   []clause.Column{{Name: "interaction_id"}},
					UpdateAll: false,
				}).
				CreateInBatches(interactions, self.Config.Relayer.StoreBatchSize).
				Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to insert Interactions")
				self.Log.WithField("interactions", interactions).Debug("Failed interactions")
				return err
			}

			// Save bundle items
			err = tx.WithContext(self.Ctx).
				Table("bundle_items").
				Clauses(clause.OnConflict{
					DoNothing: true,
					Columns:   []clause.Column{{Name: "interaction_id"}},
					UpdateAll: false,
				}).
				CreateInBatches(bundleItems, self.Config.Relayer.StoreBatchSize).
				Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to insert BundleItems")
				return err
			}

			return nil
		})
	if err != nil {
		return
	}

	// // Successfuly saved interactions
	// self.monitor.GetReport().Syncer.State.InteractionsSaved.Add(uint64(len(data)))

	// // Update saved block height
	// self.savedBlockHeight = self.finishedHeight

	// self.monitor.GetReport().Syncer.State.FinishedHeight.Store(int64(self.savedBlockHeight))

	// // Processing stops here, no need to return anything
	// out = nil
	return
}
