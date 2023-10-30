package report

import (
	"go.uber.org/atomic"
)

type RelayerErrors struct {
	SequencerPermanentParsingError       atomic.Uint64 `json:"sequencer_permanent_parsing_error"`
	SequencerBlockDownloadError          atomic.Uint64 `json:"sequencer_block_download_error"`
	SequencerPermanentBlockDownloadError atomic.Uint64 `json:"sequencer_permanent_block_download_error"`
}

type RelayerState struct {
	SequencerBlocksDownloaded     atomic.Uint64 `json:"sequencer_blocks_downloaded"`
	SequencerTransactionsReceived atomic.Uint64 `json:"sequencer_transactions_received"`
	SequencerMessagesReceived     atomic.Uint64 `json:"sequencer_messages_received"`
	SequencerTransactionsParsed   atomic.Uint64 `json:"sequencer_transactions_parsed"`

	BundleItemsSaved                         atomic.Uint64  `json:"bundle_items_saved"`
	AverageSequencerBlocksProcessedPerMinute atomic.Float64 `json:"average_blocks_processed_per_minute"`
}

type RelayerReport struct {
	State  RelayerState  `json:"state"`
	Errors RelayerErrors `json:"errors"`
}
