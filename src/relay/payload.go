package relay

import (
	"github.com/cometbft/cometbft/libs/bytes"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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

	// L1 interactions parsed from Arweave txs
	Interactions []*model.Interaction
}

type Payload struct {
	SequencerBlockHash      bytes.HexBytes
	SequencerBlockHeight    int64
	SequencerBlockTimestamp int64

	// Arweave block height used to generate sort keys for this sequencer height
	LastArweaveBlockHeight uint64

	// Raw messages from the block, only decoded
	Messages []cosmostypes.Msg

	// L2 interactions parsed from Sequencer's txs
	Interactions []*model.Interaction

	// Bundle items that will be sent to bundlr.network
	BundleItems []*model.BundleItem

	// Info about Arweave blocks
	ArweaveBlocks []*ArweaveBlock
}
