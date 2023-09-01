package relay

import (
	"errors"
	"time"

	"github.com/cometbft/cometbft/types"
	"github.com/cosmos/cosmos-sdk/client"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	proto "github.com/cosmos/gogoproto/proto"
	"github.com/warp-contracts/sequencer/app"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"

	"github.com/warp-contracts/syncer/src/utils/arweave"
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
	*task.Processor[*types.Block, types.Tx]

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

	self.Processor = task.NewProcessor[*types.Block, types.Tx](config, "store").
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

func (self *Store) process(block *types.Block) (out []types.Tx, err error) {
	self.Log.WithField("height", block.Height).Debug("Processing")
	self.finishedTimestamp = uint64(block.Time.UnixMilli())
	self.finishedHeight = uint64(block.Height)
	self.finishedBlockHash = block.LastCommitHash
	out = block.Txs
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

func (self *Store) parseInteraction(txBytes types.Tx) (out *model.Interaction, err error) {
	// Decode transaction
	tx, err := self.txConfig.TxDecoder()(txBytes)
	if err != nil {
		return
	}

	// Tx should contain one message, that is a DataItem
	dataItem, err := self.getDataItem(tx)
	if err != nil {
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
	out, err = self.parser.Parse(&dataItem.DataItem, 0, arweave.Base64String{0}, time.Now().UnixMilli(), "", "")
	if err != nil {
		self.Log.WithError(err).Error("Failed to parse interaction")
		return
	}

	return
}

func (self *Store) flush(data []types.Tx) (out []types.Tx, err error) {
	self.Log.WithField("count", len(data)).Debug("Flushing")
	if self.savedBlockHeight == self.finishedHeight && len(data) == 0 {
		// No need to flush, nothing changed
		return
	}

	self.Log.WithField("count", len(data)).Debug("Flushing interactions")
	defer self.Log.Trace("Flushing interactions done")

	// Transactions -> Interactions
	interactions := make([]*model.Interaction, 0, len(data))
	for _, txBytes := range data {
		interaction, err := self.parseInteraction(txBytes)
		if err != nil {
			self.Log.WithError(err).Error("Failed to parse interaction, skipping")
			continue
		}
		interactions = append(interactions, interaction)
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

			if len(data) == 0 {
				return nil
			}

			err = tx.WithContext(self.Ctx).
				Table("interactions").
				Clauses(clause.OnConflict{
					DoNothing: true,
					Columns:   []clause.Column{{Name: "interaction_id"}},
					UpdateAll: false,
				}).
				CreateInBatches(interactions, len(interactions)).
				Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to insert Interactions")
				self.Log.WithField("interactions", data).Debug("Failed interactions")
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
