package report

import "go.uber.org/atomic"

type TransactionDownloaderErrors struct {
	TransactionDownloadErrors atomic.Int64 `json:"block_download"`
	BlockValidationErrors     atomic.Int64 `json:"block_validation"`
	TxDownloadErrors          atomic.Int64 `json:"tx_download"`
}

type TransactionDownloaderState struct {
	SyncerCurrentHeight    atomic.Int64  `json:"syncer_current_height"`
	TransactionsDownloaded atomic.Uint64 `json:"transactions_downloaded"`
	SyncerBlocksBehind     atomic.Int64  `json:"syncer_blocks_behind"`

	AverageBlocksProcessedPerMinute       atomic.Float64 `json:"average_blocks_processed_per_minute"`
	AverageTransactionDownloadedPerMinute atomic.Float64 `json:"average_transactions_downloaded_per_minute"`
}

type TransactionDownloaderReport struct {
	State  TransactionDownloaderState  `json:"state"`
	Errors TransactionDownloaderErrors `json:"errors"`
}
