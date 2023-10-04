package relay

import (
	"github.com/warp-contracts/sequencer/x/sequencer/types"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/model"
)

type ArweaveBlock struct {
	// Arweave Blocks with L1 transactions, parsed from Sequencer's txs
	Message *types.MsgArweaveBlock

	// Corresponding block downloaded from Arweave
	Block *arweave.Block

	// Transactions from the block
	Transactions []*arweave.Transaction
}

type Payload struct {
	SequencerBlockHash      string
	SequencerBlockHeight    int64
	SequencerBlockTimestamp int64

	// L2 interactions parsed from Sequencer's txs
	Interactions []*model.Interaction

	// Bundle items that will be sent to bundlr.network
	BundleItems []*model.BundleItem

	// Info about Arweave blocks
	ArweaveBlocks []*ArweaveBlock
}
