package report

import "go.uber.org/atomic"

type BlockDownloaderErrors struct {
	BlockDownloadErrors   atomic.Int64 `json:"block_download"`
	BlockValidationErrors atomic.Int64 `json:"block_validation"`
	TxDownloadErrors      atomic.Int64 `json:"tx_download"`
}

type BlockDownloaderState struct {
	SyncerCurrentHeight    atomic.Int64  `json:"syncer_current_height"`
	TransactionsDownloaded atomic.Uint64 `json:"transactions_downloaded"`
	BlocksBehind           atomic.Int64  `json:"syncer_blocks_behind"`

	AverageBlocksProcessedPerMinute       atomic.Float64 `json:"average_blocks_processed_per_minute"`
	AverageTransactionDownloadedPerMinute atomic.Float64 `json:"average_transactions_downloaded_per_minute"`
}

type BlockDownloaderReport struct {
	State  BlockDownloaderState  `json:"state"`
	Errors BlockDownloaderErrors `json:"errors"`
}
