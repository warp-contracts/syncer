package report

import (
	"go.uber.org/atomic"
)

type SyncerErrors struct {
	DbInteractionInsert               atomic.Int64 `json:"db_interaction"`
	DbLastTransactionBlockHeightError atomic.Int64 `json:"db_last_tx_block_height"`
	TxValidationErrors                atomic.Int64 `json:"tx_validation"`
	TxDownloadErrors                  atomic.Int64 `json:"tx_download"`
	BlockValidationErrors             atomic.Int64 `json:"block_validation"`
	BlockDownloadErrors               atomic.Int64 `json:"block_download"`
	PeerDownloadErrors                atomic.Int64 `json:"peer_download"`
}

type SyncerState struct {
	StartTimestamp       atomic.Int64  `json:"start_timestamp"`
	UpForSeconds         atomic.Uint64 `json:"up_for_seconds"`
	SyncerBlocksBehind   atomic.Int64  `json:"syncer_blocks_behind"`
	SyncerCurrentHeight  atomic.Int64  `json:"syncer_current_height"`
	SyncerFinishedHeight atomic.Int64  `json:"syncer_finished_height"`

	AverageBlocksProcessedPerMinute       atomic.Float64 `json:"average_blocks_processed_per_minute"`
	AverageTransactionDownloadedPerMinute atomic.Float64 `json:"average_transactions_downloaded_per_minute"`
	AverageInteractionsSavedPerMinute     atomic.Float64 `json:"average_interactions_saved_per_minute"`

	PeersBlacklisted atomic.Uint64 `json:"peers_blacklisted"`
	NumPeers         atomic.Uint64 `json:"num_peers"`

	TransactionsDownloaded   atomic.Uint64 `json:"transactions_downloaded"`
	InteractionsSaved        atomic.Uint64 `json:"interactions_saved"`
	FailedInteractionParsing atomic.Uint64 `json:"failed_interaction_parsing"`
	NumWatchdogRestarts      atomic.Uint64 `json:"num_watchdog_restarts"`
}

type SyncerReport struct {
	State  SyncerState  `json:"state"`
	Errors SyncerErrors `json:"errors"`
}
