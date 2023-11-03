package report

import (
	"go.uber.org/atomic"
)

type RelayerErrors struct {
	SequencerPermanentParsingError       atomic.Uint64 `json:"sequencer_permanent_parsing_error"`
	SequencerBlockDownloadError          atomic.Uint64 `json:"sequencer_block_download_error"`
	SequencerPermanentBlockDownloadError atomic.Uint64 `json:"sequencer_permanent_block_download_error"`

	PersistentSequencerFailedParsing atomic.Uint64 `json:"persistent_sequencer_failed_parsing"`
	PersistentArweaveFailedParsing   atomic.Uint64 `json:"persistent_arweave_failed_parsing"`
	DbError                          atomic.Uint64 `json:"db_error"`
}

type RelayerState struct {
	SequencerBlocksDownloaded     atomic.Uint64 `json:"sequencer_blocks_downloaded"`
	SequencerTransactionsReceived atomic.Uint64 `json:"sequencer_transactions_received"`
	SequencerMessagesReceived     atomic.Uint64 `json:"sequencer_messages_received"`
	SequencerTransactionsParsed   atomic.Uint64 `json:"sequencer_transactions_parsed"`
	SequencerFinishedHeight       atomic.Int64  `json:"sequencer_finished_height"`

	BundleItemsSaved                         atomic.Uint64  `json:"bundle_items_saved"`
	AverageSequencerBlocksProcessedPerMinute atomic.Float64 `json:"average_blocks_processed_per_minute"`

	// Store
	ArwaeveFinishedHeight             atomic.Int64   `json:"arweave_finished_height"`
	AverageInteractionsSavedPerMinute atomic.Float64 `json:"average_interactions_saved_per_minute"`
	L1InteractionsSaved               atomic.Uint64  `json:"l1_interactions_saved"`
	L2InteractionsSaved               atomic.Uint64  `json:"l2_interactions_saved"`
	InteractionsSaved                 atomic.Uint64  `json:"interactions_saved"`
	FailedInteractionParsing          atomic.Uint64  `json:"failed_interaction_parsing"`
}

type RelayerReport struct {
	State  RelayerState  `json:"state"`
	Errors RelayerErrors `json:"errors"`
}
