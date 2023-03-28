package report

type Report struct {
	Run                   *RunReport                   `json:"run,omitempty"`
	Peer                  *PeerReport                  `json:"peer,omitempty"`
	Syncer                *SyncerReport                `json:"syncer,omitempty"`
	Bundler               *BundlerReport               `json:"bundler,omitempty"`
	Checker               *CheckerReport               `json:"checker,omitempty"`
	NetworkInfo           *NetworkInfoReport           `json:"network_info,omitempty"`
	BlockMonitor          *BlockMonitorReport          `json:"block_monitor,omitempty"`
	BlockDownloader       *BlockDownloaderReport       `json:"block_downloader,omitempty"`
	TransactionDownloader *TransactionDownloaderReport `json:"transaction_downloader,omitempty"`
}
