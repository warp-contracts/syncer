package report

import "go.uber.org/atomic"

type WarpySyncerErrors struct {
	BlockDownloaderFailures              atomic.Uint64 `json:"block_downloader_failures"`
	SyncerDeltaCheckTxFailures           atomic.Uint64 `json:"syncer_delta_check_tx_failures"`
	SyncerDeltaProcessTxPermanentError   atomic.Uint64 `json:"syncer_delta_process_tx_permanent_error"`
	SyncerDepositProcessTxPermanentError atomic.Uint64 `json:"syncer_deposit_process_tx_permanent_error"`
	SyncerDepositCheckTxFailures         atomic.Uint64 `json:"syncer_deposit_check_tx_failures"`
	WriterFailures                       atomic.Uint64 `json:"writer_failures"`
	StoreGetLastStateFailure             atomic.Uint64 `json:"store_get_last_state_failure"`
	StoreSaveLastStateFailure            atomic.Uint64 `json:"store_save_last_state_failure"`
	PollerDepositFetchError              atomic.Uint64 `json:"poller_deposit_fetch_error"`
	StoreDepositFailures                 atomic.Uint64 `json:"store_deposit_failures"`
}

type WarpySyncerState struct {
	BlockDownloaderCurrentHeight   atomic.Int64 `json:"block_downloader_current_height"`
	SyncerDeltaTxsProcessed        atomic.Int64 `json:"syncer_delta_txs_processed"`
	SyncerDeltaBlocksProcessed     atomic.Int64 `json:"syncer_delta_blocks_processed"`
	SyncerDepositTxsProcessed      atomic.Int64 `json:"syncer_deposit_txs_processed"`
	SyncerDepositBlocksProcessed   atomic.Int64 `json:"syncer_deposit_blocks_processed"`
	WriterInteractionsToWarpy      atomic.Int64 `json:"writer_interactions_to_warpy"`
	StoreLastSyncedBlockHeight     atomic.Int64 `json:"store_last_synced_block_height"`
	PollerDepositAssetsFromSelects atomic.Int64 `json:"poller_deposit_assets_from_selects"`
	StoreDepositRecordsSaved       atomic.Int64 `json:"store_deposit_records_saved"`
}

type WarpySyncerReport struct {
	State  WarpySyncerState  `json:"state"`
	Errors WarpySyncerErrors `json:"errors"`
}
