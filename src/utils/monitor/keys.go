package monitor

type Kind string

var (
	DbInteraction                     = "db_interaction"
	DbLastTransactionBlockHeightError = "db_last_tx_block_height"
	TxValidationErrors                = "tx_validation"
	TxDownloadErrors                  = "tx_download"
	BlockValidationErrors             = "block_validation"
	BlockDownloadErrors               = "block_download"
	PeerDownloadErrors                = "peer_download"
	NetworkInfoDownloadErrors         = "network_info_download"
)
