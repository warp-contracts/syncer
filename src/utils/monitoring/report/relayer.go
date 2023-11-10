package report

import (
	"go.uber.org/atomic"
)

type RelayerErrors struct {
	SequencerPermanentParsingError       atomic.Uint64 `json:"sequencer_permanent_parsing_error"`
	SequencerBlockDownloadError          atomic.Uint64 `json:"sequencer_block_download_error"`
	SequencerPermanentBlockDownloadError atomic.Uint64 `json:"sequencer_permanent_block_download_error"`

	PersistentArweaveFailedParsing atomic.Uint64 `json:"persistent_arweave_failed_parsing"`
	DbError                        atomic.Uint64 `json:"db_error"`
}

type RelayerState struct {
	// Source
	SequencerBlocksDownloaded                atomic.Uint64  `json:"sequencer_blocks_downloaded"`
	SequencerBlocksStreamed                  atomic.Uint64  `json:"sequencer_blocks_streamed"`
	SequencerBlocksCatchedUp                 atomic.Uint64  `json:"sequencer_blocks_catched_up"`
	AverageSequencerBlocksProcessedPerMinute atomic.Float64 `json:"average_sequencer_blocks_processed_per_minute"`

	// Decoder
	SequencerTransactionsDecoded atomic.Uint64 `json:"sequencer_transactions_decoded"`

	// Parser
	SequencerTransactionsParsed atomic.Uint64 `json:"sequencer_transactions_parsed"`

	// Store
	ArwaeveFinishedHeight   atomic.Int64  `json:"arweave_finished_height"`
	SequencerFinishedHeight atomic.Int64  `json:"sequencer_finished_height"`
	BundleItemsSaved        atomic.Uint64 `json:"bundle_items_saved"`

	AverageInteractionsSavedPerMinute atomic.Float64 `json:"average_interactions_saved_per_minute"`
	L1InteractionsSaved               atomic.Uint64  `json:"l1_interactions_saved"`
	L2InteractionsSaved               atomic.Uint64  `json:"l2_interactions_saved"`
	InteractionsSaved                 atomic.Uint64  `json:"interactions_saved"`
}

type RelayerReport struct {
	State  RelayerState  `json:"state"`
	Errors RelayerErrors `json:"errors"`
}
