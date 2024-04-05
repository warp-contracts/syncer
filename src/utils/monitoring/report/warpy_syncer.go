package report

import "go.uber.org/atomic"

type WarpySyncerErrors struct {
	BlockDownloaderFailures                atomic.Uint64 `json:"block_downloader_failures"`
	SyncerDeltaCheckTxFailures             atomic.Uint64 `json:"syncer_delta_check_tx_failures"`
	SyncerDeltaProcessTxPermanentError     atomic.Uint64 `json:"syncer_delta_process_tx_permanent_error"`
	SyncerSommelierProcessTxPermanentError atomic.Uint64 `json:"syncer_sommelier_process_tx_permanent_error"`
	SyncerSommelierCheckTxFailures         atomic.Uint64 `json:"syncer_sommelier_check_tx_failures"`
	WriterFailures                         atomic.Uint64 `json:"writer_failures"`
	StoreGetLastStateFailure               atomic.Uint64 `json:"store_get_last_state_failure"`
	StoreSaveLastStateFailure              atomic.Uint64 `json:"store_save_last_state_failure"`
	PollerSommelierFetchError              atomic.Uint64 `json:"poller_sommelier_fetch_error"`
	StoreSommelierFailures                 atomic.Uint64 `json:"store_sommelier_failures"`
}

type WarpySyncerState struct {
	BlockDownloaderCurrentHeight     atomic.Int64 `json:"block_downloader_current_height"`
	SyncerDeltaTxsProcessed          atomic.Int64 `json:"syncer_delta_txs_processed"`
	SyncerDeltaBlocksProcessed       atomic.Int64 `json:"syncer_delta_blocks_processed"`
	SyncerSommelierTxsProcessed      atomic.Int64 `json:"syncer_sommelier_txs_processed"`
	SyncerSommelierBlocksProcessed   atomic.Int64 `json:"syncer_sommelier_blocks_processed"`
	WriterInteractionsToWarpy        atomic.Int64 `json:"writer_interactions_to_warpy"`
	StoreLastSyncedBlockHeight       atomic.Int64 `json:"store_last_synced_block_height"`
	PollerSommelierAssetsFromSelects atomic.Int64 `json:"poller_sommelier_assets_from_selects"`
	StoreSommelierRecordsSaved       atomic.Int64 `json:"store_sommelier_records_saved"`
}

type WarpySyncerReport struct {
	State  WarpySyncerState  `json:"state"`
	Errors WarpySyncerErrors `json:"errors"`
}
