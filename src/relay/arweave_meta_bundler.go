package relay

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strconv"

	"github.com/jackc/pgtype"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
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
func (self *ArweaveMetaBundler) createDataItem(arweaveBlock *ArweaveBlock, idx int) (out *bundlr.BundleItem, err error) {
	interaction := arweaveBlock.Interactions[idx]
	info := arweaveBlock.Message.Transactions[idx]

	out = new(bundlr.BundleItem)
	out.SignatureType = bundlr.SignatureTypeArweave

	// Prevent accidental replying of the same sort key by using the anchor field
	hash := sha256.Sum256([]byte(interaction.SortKey))
	out.Anchor = hash[:]

	// Tags
	out.Tags = bundlr.Tags{
		{Name: "Bundle-Format", Value: "binary"},
		{Name: "Bundle-Version", Value: "2.0.0"},
		{Name: "App-Name", Value: "Warp"},
		{Name: "Action", Value: "WarpInteraction"},
		// Tags common with L2
		{Name: "Sequencer", Value: "RedStone"},
		{Name: "Sequencer-Contract", Value: interaction.ContractId},
		{Name: "Sequencer-Tx-Id", Value: interaction.InteractionId.Base64()},
		{Name: "Sequencer-Sort-Key", Value: interaction.SortKey},
		{Name: "Sequencer-Random", Value: base64.RawURLEncoding.EncodeToString(info.Random)},
	}

	// Set previous sort key only if present
	if interaction.LastSortKey.Status == pgtype.Present {
		out.Tags = append(out.Tags, bundlr.Tag{
			Name:  "Sequencer-Prev-Sort-Key",
			Value: interaction.LastSortKey.String,
		})
	}

	// Sign
	err = out.Sign(self.signer)
	if err != nil {
		return
	}

	return
}

func (self *ArweaveMetaBundler) createMetaDataItem(payload *Payload, arweaveBlock *ArweaveBlock, items []*bundlr.BundleItem) (out *bundlr.BundleItem, err error) {
	out = new(bundlr.BundleItem)
	out.SignatureType = bundlr.SignatureTypeArweave

	// Prevent accidental replying of the same
	hash := sha256.Sum256(payload.SequencerBlockHash.Bytes())
	out.Anchor = hash[:]

	err = out.NestBundles(items)
	if err != nil {
		return
	}

	out.Tags = bundlr.Tags{
		{Name: "Bundle-Format", Value: "binary"}, {Name: "Bundle-Version", Value: "2.0.0"}, {Name: "App-Name", Value: "Warp"},
		// FIXME: Is this Action OK?
		{Name: "Action", Value: "WarpMetaData"},
		// Tags common with L2
		{Name: "Sequencer", Value: "RedStone"},
		{Name: "Sequencer-Arweave-Height", Value: strconv.FormatUint(arweaveBlock.Message.BlockInfo.Height, 10)},
		{Name: "Sequencer-Arweave-Timestamp", Value: strconv.FormatUint(arweaveBlock.Message.BlockInfo.Timestamp, 10)},
		{Name: "Sequencer-Arweave-Hash", Value: arweaveBlock.Message.BlockInfo.Hash},
		{Name: "Sequencer-Height", Value: strconv.FormatInt(payload.SequencerBlockHeight, 10)},
		{Name: "Sequencer-Timestamp", Value: strconv.FormatInt(payload.SequencerBlockTimestamp, 10)},
		{Name: "Sequencer-Hash", Value: base64.RawURLEncoding.EncodeToString(payload.SequencerBlockHash.Bytes())},
	}

	// Sign
	err = out.Sign(self.signer)
	if err != nil {
		return
	}

	return
}

func (self *ArweaveMetaBundler) fill(payload *Payload) (err error) {
	for blockIdx, arweaveBlock := range payload.ArweaveBlocks {
		items := make([]*bundlr.BundleItem, len(arweaveBlock.Interactions))
		for i := range arweaveBlock.Interactions {
			items[i], err = self.createDataItem(arweaveBlock, i)
			if err != nil {
				return
			}
		}

		payload.ArweaveBlocks[blockIdx].MetaInfoDataItem, err = self.createMetaDataItem(payload, arweaveBlock, items)
		if err != nil {
			return
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
