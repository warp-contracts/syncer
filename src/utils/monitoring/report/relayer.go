package report

import (
	"go.uber.org/atomic"
)

type RelayerErrors struct {
	SequencerBlockDownloadError          atomic.Uint64 `json:"sequencer_block_download_error"`
	SequencerPermanentBlockDownloadError atomic.Uint64 `json:"sequencer_permanent_block_download_error"`
}

type RelayerState struct {
	BlocksReceived       atomic.Uint64 `json:"blocks_received"`
	TransactionsReceived atomic.Uint64 `json:"transactions_received"`
	MessagesReceived     atomic.Uint64 `json:"messages_received"`

	BundleItemsSaved atomic.Uint64 `json:"bundle_items_saved"`
}

type RelayerReport struct {
	State  RelayerState  `json:"state"`
	Errors RelayerErrors `json:"errors"`
}
