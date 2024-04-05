package report

type Report struct {
	Run                   *RunReport                   `json:"run,omitempty"`
	Peer                  *PeerReport                  `json:"peer,omitempty"`
	Syncer                *SyncerReport                `json:"syncer,omitempty"`
	Contractor            *ContractorReport            `json:"contractor,omitempty"`
	Bundler               *BundlerReport               `json:"bundler,omitempty"`
	Sender                *SenderReport                `json:"sender,omitempty"`
	Checker               *CheckerReport               `json:"checker,omitempty"`
	Forwarder             *ForwarderReport             `json:"forwarder,omitempty"`
	Relayer               *RelayerReport               `json:"relayer,omitempty"`
	Gateway               *GatewayReport               `json:"gateway,omitempty"`
	Interactor            *InteractorReport            `json:"interactor,omitempty"`
	NetworkInfo           *NetworkInfoReport           `json:"network_info,omitempty"`
	BlockMonitor          *BlockMonitorReport          `json:"block_monitor,omitempty"`
	BlockDownloader       *BlockDownloaderReport       `json:"block_downloader,omitempty"`
	TransactionDownloader *TransactionDownloaderReport `json:"transaction_downloader,omitempty"`
	RedisPublishers       []RedisPublisherReport       `json:"redis_publishers,omitempty"`
	AppSyncPublisher      *AppSyncPublisherReport      `json:"appsync_publisher,omitempty"`
	Evolver               *EvolverReport               `json:"evolver,omitempty"`
	WarpySyncer           *WarpySyncerReport           `json:"warpy_syncer,omitempty"`
}
