package report

import "go.uber.org/atomic"

type RedstoneTxSyncerErrors struct {
	BlockDownloaderFailures         atomic.Uint64 `json:"block_downloader_failures"`
	SyncerWriteInteractionsFailures atomic.Uint64 `json:"syncer_write_interactions_failures"`
	SyncerUpdateSyncStateFailures   atomic.Uint64 `json:"syncer_update_sync_state_failures"`
}

type RedstoneTxSyncerState struct {
	BlockDownloaderCurrentHeight atomic.Int64 `json:"block_downloader_current_height"`
	SyncerTxsProcessed           atomic.Int64 `json:"syncer_txs_processed"`
	SyncerBlocksProcessed        atomic.Int64 `json:"syncer_blocks_processed"`
	SyncerInteractionsToWarpy    atomic.Int64 `json:"syncer_interactions_to_warpy"`
}

type RedstoneTxSyncerReport struct {
	State  RedstoneTxSyncerState  `json:"state"`
	Errors RedstoneTxSyncerErrors `json:"errors"`
}
