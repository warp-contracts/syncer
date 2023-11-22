package relay

import (
	"crypto/sha256"
	"errors"

	"github.com/jackc/pgtype"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
)

// Uses MsgArweaveBlock messages to create one nested bundle
// Contains data assigned by the sequencer to interactions in one Arweave block
type ArweaveMetaBundler struct {
	*task.Task

	monitor monitoring.Monitor
	signer  *bundlr.ArweaveSigner

	input  <-chan *Payload
	Output chan *Payload
}

func NewArweaveMetaBundler(config *config.Config) (self *ArweaveMetaBundler) {
	self = new(ArweaveMetaBundler)

	self.Output = make(chan *Payload)

	var err error
	self.signer, err = bundlr.NewArweaveSigner(config.Bundlr.Wallet)
	if err != nil {
		self.Log.WithError(err).Panic("Failed to create bundlr signer")
	}

	self.Task = task.NewTask(config, "arweave_meta_bundler").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *ArweaveMetaBundler) WithMonitor(monitor monitoring.Monitor) *ArweaveMetaBundler {
	self.monitor = monitor
	return self
}

func (self *ArweaveMetaBundler) WithInputChannel(v <-chan *Payload) *ArweaveMetaBundler {
	self.input = v
	return self
}

// Create a data item with data assigned to L1 interaction
func (self *ArweaveMetaBundler) createDataItem(payload *Payload, arweaveBlock *ArweaveBlock, idx int) (out *bundlr.BundleItem, err error) {
	interaction := arweaveBlock.Interactions[idx]
	info := arweaveBlock.Message.Transactions[idx]

	out = new(bundlr.BundleItem)
	out.SignatureType = bundlr.SignatureTypeArweave

	// Prevent accidental replying of the same sort key by using the anchor field
	hash := sha256.Sum256([]byte(interaction.SortKey))
	out.Anchor = hash[:]

	// Tags
	out.Tags = getTags(payload, "Arweave", self.Config.Relayer.Environment, interaction, info.Random)

	// Sign
	err = out.Sign(self.signer)
	if err != nil {
		return
	}

	return
}

func (self *ArweaveMetaBundler) fill(payload *Payload) (err error) {
	for blockIdx, arweaveBlock := range payload.ArweaveBlocks {
		for i := range arweaveBlock.Interactions {
			var item *bundlr.BundleItem
			item, err = self.createDataItem(payload, arweaveBlock, i)
			if err != nil {
				return
			}

			var dataItemBytes []byte
			dataItemBytes, err = item.Marshal()
			if err != nil {
				return
			}

			// Create a DataItems that get inserted to the database
			payload.ArweaveBlocks[blockIdx].MetaInfoDataItems = append(payload.ArweaveBlocks[blockIdx].MetaInfoDataItems,
				&model.DataItem{
					DataItemID:  item.Id.Base64(),
					State:       model.BundleStatePending,
					DataItem:    pgtype.Bytea{Bytes: dataItemBytes, Status: pgtype.Present},
					BlockHeight: pgtype.Int8{Status: pgtype.Null},
					Response:    pgtype.JSONB{Status: pgtype.Null},
					Service:     pgtype.Text{Status: pgtype.Null},
				},
			)
		}
	}

	return
}

func (self *ArweaveMetaBundler) run() (err error) {
	for payload := range self.input {
		err = self.fill(payload)
		if err != nil {
			if self.IsStopping.Load() {
				// Neglect, we're stopping anyway
				return nil
			}

			self.Log.WithField("sequencer_height", payload.SequencerBlockHeight).WithError(err).Error("Failed to create arweave block bundle")

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
