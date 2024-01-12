package report

import "go.uber.org/atomic"

type RedstoneTxSyncerErrors struct {
	BlockDownloaderFailures               atomic.Uint64 `json:"block_downloader_failures"`
	SyncerWriteInteractionFailures        atomic.Uint64 `json:"syncer_write_interaction_failures"`
	SyncerWriteInteractionsPermanentError atomic.Uint64 `json:"syncer_write_interaction_permanent_error"`
	StoreGetLastStateFailure              atomic.Uint64 `json:"store_get_last_state_failure"`
	StoreSaveLastStateFailure             atomic.Uint64 `json:"store_save_last_state_failure"`
}

type RedstoneTxSyncerState struct {
	BlockDownloaderCurrentHeight atomic.Int64 `json:"block_downloader_current_height"`
	SyncerTxsProcessed           atomic.Int64 `json:"syncer_txs_processed"`
	SyncerBlocksProcessed        atomic.Int64 `json:"syncer_blocks_processed"`
	SyncerInteractionsToWarpy    atomic.Int64 `json:"syncer_interactions_to_warpy"`
	StoreLastSyncedBlockHeight   atomic.Int64 `json:"store_last_synced_block_height"`
}

type RedstoneTxSyncerReport struct {
	State  RedstoneTxSyncerState  `json:"state"`
	Errors RedstoneTxSyncerErrors `json:"errors"`
}
