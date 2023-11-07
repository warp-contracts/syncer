package monitor_relayer

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/warp-contracts/syncer/src/utils/config"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds        *prometheus.Desc
	NumWatchdogRestarts *prometheus.Desc

	// Network
	NetworkInfoDownloadErrors *prometheus.Desc
	// ArweaveCurrentHeight            *prometheus.Desc
	// ArweaveLastNetworkInfoTimestamp *prometheus.Desc

	// BlockDownloader
	BlockDownloadErrors             *prometheus.Desc
	BlockValidationErrors           *prometheus.Desc
	BlockCurrentHeight              *prometheus.Desc
	BlocksBehind                    *prometheus.Desc
	AverageBlocksProcessedPerMinute *prometheus.Desc

	// TransactionDownloader
	TransactionsDownloaded                *prometheus.Desc
	AverageTransactionDownloadedPerMinute *prometheus.Desc
	TxPermanentDownloadErrors             *prometheus.Desc
	TxValidationErrors                    *prometheus.Desc
	TxDownloadErrors                      *prometheus.Desc

	// PeerMonitor
	PeersBlacklisted   *prometheus.Desc
	NumPeers           *prometheus.Desc
	PeerDownloadErrors *prometheus.Desc

	// Relayer
	SequencerPermanentParsingError           *prometheus.Desc
	SequencerBlockDownloadError              *prometheus.Desc
	SequencerPermanentBlockDownloadError     *prometheus.Desc
	PersistentSequencerFailedParsing         *prometheus.Desc
	PersistentArweaveFailedParsing           *prometheus.Desc
	DbError                                  *prometheus.Desc
	SequencerBlocksDownloaded                *prometheus.Desc
	SequencerBlocksStreamed                  *prometheus.Desc
	SequencerBlocksCatchedUp                 *prometheus.Desc
	AverageSequencerBlocksProcessedPerMinute *prometheus.Desc
	SequencerTransactionsParsed              *prometheus.Desc
	ArwaeveFinishedHeight                    *prometheus.Desc
	SequencerFinishedHeight                  *prometheus.Desc
	BundleItemsSaved                         *prometheus.Desc
	AverageInteractionsSavedPerMinute        *prometheus.Desc
	L1InteractionsSaved                      *prometheus.Desc
	L2InteractionsSaved                      *prometheus.Desc
	InteractionsSaved                        *prometheus.Desc
}

func NewCollector(config *config.Config) *Collector {
	collector := &Collector{
		// Run
		UpForSeconds:        prometheus.NewDesc("up_for_seconds", "", nil, nil),
		NumWatchdogRestarts: prometheus.NewDesc("num_watchdog_restarts", "", nil, nil),

		// NetworkMonitor
		NetworkInfoDownloadErrors: prometheus.NewDesc("error_network_info_download", "", nil, nil),

		// BlockDownloader - ok
		BlockDownloadErrors:             prometheus.NewDesc("error_block_download", "", nil, nil),
		BlockValidationErrors:           prometheus.NewDesc("error_block_validation", "", nil, nil),
		BlockCurrentHeight:              prometheus.NewDesc("block_current_height", "", nil, nil),
		BlocksBehind:                    prometheus.NewDesc("blocks_behind", "", nil, nil),
		AverageBlocksProcessedPerMinute: prometheus.NewDesc("average_blocks_processed_per_minute", "", nil, nil),

		// TransactionDownloader - ok
		TransactionsDownloaded:                prometheus.NewDesc("transactions_downloaded", "", nil, nil),
		AverageTransactionDownloadedPerMinute: prometheus.NewDesc("average_transactions_downloaded_per_minute", "", nil, nil),
		TxValidationErrors:                    prometheus.NewDesc("error_tx_validation", "", nil, nil),
		TxDownloadErrors:                      prometheus.NewDesc("error_tx_download", "", nil, nil),
		TxPermanentDownloadErrors:             prometheus.NewDesc("error_tx_permanent_download", "", nil, nil),

		// PeerMonitor
		PeersBlacklisted:   prometheus.NewDesc("peers_blacklisted", "", nil, nil),
		NumPeers:           prometheus.NewDesc("num_peers", "", nil, nil),
		PeerDownloadErrors: prometheus.NewDesc("error_peer_download", "", nil, nil),

		// Relayer
		SequencerPermanentParsingError:           prometheus.NewDesc("sequencer_permanent_parsing_error", "", nil, nil),
		SequencerBlockDownloadError:              prometheus.NewDesc("sequencer_block_download_error", "", nil, nil),
		SequencerPermanentBlockDownloadError:     prometheus.NewDesc("sequencer_permanent_block_download_error", "", nil, nil),
		PersistentSequencerFailedParsing:         prometheus.NewDesc("persistent_sequencer_failed_parsing", "", nil, nil),
		PersistentArweaveFailedParsing:           prometheus.NewDesc("persistent_arweave_failed_parsing", "", nil, nil),
		DbError:                                  prometheus.NewDesc("db_error", "", nil, nil),
		SequencerBlocksDownloaded:                prometheus.NewDesc("sequencer_blocks_downloaded", "", nil, nil),
		SequencerBlocksStreamed:                  prometheus.NewDesc("sequencer_blocks_streamed", "", nil, nil),
		SequencerBlocksCatchedUp:                 prometheus.NewDesc("sequencer_blocks_catched_up", "", nil, nil),
		AverageSequencerBlocksProcessedPerMinute: prometheus.NewDesc("average_sequencer_blocks_processed_per_minute", "", nil, nil),
		SequencerTransactionsParsed:              prometheus.NewDesc("sequencer_transactions_parsed", "", nil, nil),
		ArwaeveFinishedHeight:                    prometheus.NewDesc("arweave_finished_height", "", nil, nil),
		SequencerFinishedHeight:                  prometheus.NewDesc("sequencer_finished_height", "", nil, nil),
		BundleItemsSaved:                         prometheus.NewDesc("bundle_items_saved", "", nil, nil),
		AverageInteractionsSavedPerMinute:        prometheus.NewDesc("average_interactions_saved_per_minute", "", nil, nil),
		L1InteractionsSaved:                      prometheus.NewDesc("l1_interactions_saved", "", nil, nil),
		L2InteractionsSaved:                      prometheus.NewDesc("l2_interactions_saved", "", nil, nil),
		InteractionsSaved:                        prometheus.NewDesc("interactions_saved", "", nil, nil),
	}

	return collector
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Run
	ch <- self.UpForSeconds
	ch <- self.NumWatchdogRestarts

	// NetworkInfo
	ch <- self.NetworkInfoDownloadErrors

	// BlockDownloader
	ch <- self.BlockValidationErrors
	ch <- self.BlockDownloadErrors
	ch <- self.BlockCurrentHeight
	ch <- self.BlocksBehind
	ch <- self.AverageBlocksProcessedPerMinute

	// TransactionDownloader
	ch <- self.TransactionsDownloaded
	ch <- self.AverageTransactionDownloadedPerMinute
	ch <- self.TxValidationErrors
	ch <- self.TxDownloadErrors
	ch <- self.TxPermanentDownloadErrors

	// PeerMonitor
	ch <- self.PeersBlacklisted
	ch <- self.NumPeers
	ch <- self.PeerDownloadErrors

	// Relayer
	ch <- self.SequencerPermanentParsingError
	ch <- self.SequencerBlockDownloadError
	ch <- self.SequencerPermanentBlockDownloadError
	ch <- self.PersistentSequencerFailedParsing
	ch <- self.PersistentArweaveFailedParsing
	ch <- self.DbError
	ch <- self.SequencerBlocksDownloaded
	ch <- self.SequencerBlocksStreamed
	ch <- self.SequencerBlocksCatchedUp
	ch <- self.AverageSequencerBlocksProcessedPerMinute
	ch <- self.SequencerTransactionsParsed
	ch <- self.ArwaeveFinishedHeight
	ch <- self.SequencerFinishedHeight
	ch <- self.BundleItemsSaved
	ch <- self.AverageInteractionsSavedPerMinute
	ch <- self.L1InteractionsSaved
	ch <- self.L2InteractionsSaved
	ch <- self.InteractionsSaved
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumWatchdogRestarts, prometheus.CounterValue, float64(self.monitor.Report.Run.Errors.NumWatchdogRestarts.Load()))

	// NetworkInfo
	ch <- prometheus.MustNewConstMetric(self.NetworkInfoDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.NetworkInfo.Errors.NetworkInfoDownloadErrors.Load()))

	// BlockDownloader
	ch <- prometheus.MustNewConstMetric(self.BlockValidationErrors, prometheus.CounterValue, float64(self.monitor.Report.BlockDownloader.Errors.BlockValidationErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlockDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.BlockDownloader.Errors.BlockDownloadErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlockCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.BlockDownloader.State.CurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlocksBehind, prometheus.GaugeValue, float64(self.monitor.Report.BlockDownloader.State.BlocksBehind.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageBlocksProcessedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.BlockDownloader.State.AverageBlocksProcessedPerMinute.Load()))

	// TransactionDownloader
	ch <- prometheus.MustNewConstMetric(self.TransactionsDownloaded, prometheus.CounterValue, float64(self.monitor.Report.TransactionDownloader.State.TransactionsDownloaded.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageTransactionDownloadedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.TransactionDownloader.State.AverageTransactionDownloadedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.TxValidationErrors, prometheus.CounterValue, float64(self.monitor.Report.TransactionDownloader.Errors.Validation.Load()))
	ch <- prometheus.MustNewConstMetric(self.TxPermanentDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.TransactionDownloader.Errors.PermanentDownloadFailure.Load()))
	ch <- prometheus.MustNewConstMetric(self.TxDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.TransactionDownloader.Errors.Download.Load()))

	// PeerMonitor
	ch <- prometheus.MustNewConstMetric(self.PeersBlacklisted, prometheus.GaugeValue, float64(self.monitor.Report.Peer.State.PeersBlacklisted.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumPeers, prometheus.GaugeValue, float64(self.monitor.Report.Peer.State.NumPeers.Load()))
	ch <- prometheus.MustNewConstMetric(self.PeerDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Peer.Errors.PeerDownloadErrors.Load()))

	// Relayer
	ch <- prometheus.MustNewConstMetric(self.SequencerPermanentParsingError, prometheus.CounterValue, float64(self.monitor.Report.Relayer.Errors.SequencerPermanentParsingError.Load()))
	ch <- prometheus.MustNewConstMetric(self.SequencerBlockDownloadError, prometheus.CounterValue, float64(self.monitor.Report.Relayer.Errors.SequencerBlockDownloadError.Load()))
	ch <- prometheus.MustNewConstMetric(self.SequencerPermanentBlockDownloadError, prometheus.CounterValue, float64(self.monitor.Report.Relayer.Errors.SequencerPermanentBlockDownloadError.Load()))
	ch <- prometheus.MustNewConstMetric(self.PersistentArweaveFailedParsing, prometheus.CounterValue, float64(self.monitor.Report.Relayer.Errors.PersistentArweaveFailedParsing.Load()))
	ch <- prometheus.MustNewConstMetric(self.DbError, prometheus.CounterValue, float64(self.monitor.Report.Relayer.Errors.DbError.Load()))
	ch <- prometheus.MustNewConstMetric(self.SequencerBlocksDownloaded, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.SequencerBlocksDownloaded.Load()))
	ch <- prometheus.MustNewConstMetric(self.SequencerBlocksStreamed, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.SequencerBlocksStreamed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SequencerBlocksCatchedUp, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.SequencerBlocksCatchedUp.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageSequencerBlocksProcessedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.Relayer.State.AverageSequencerBlocksProcessedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.SequencerTransactionsParsed, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.SequencerTransactionsParsed.Load()))
	ch <- prometheus.MustNewConstMetric(self.ArwaeveFinishedHeight, prometheus.GaugeValue, float64(self.monitor.Report.Relayer.State.ArwaeveFinishedHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SequencerFinishedHeight, prometheus.GaugeValue, float64(self.monitor.Report.Relayer.State.SequencerFinishedHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundleItemsSaved, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.BundleItemsSaved.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageInteractionsSavedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.Relayer.State.AverageInteractionsSavedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.L1InteractionsSaved, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.L1InteractionsSaved.Load()))
	ch <- prometheus.MustNewConstMetric(self.L2InteractionsSaved, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.L2InteractionsSaved.Load()))
	ch <- prometheus.MustNewConstMetric(self.InteractionsSaved, prometheus.CounterValue, float64(self.monitor.Report.Relayer.State.InteractionsSaved.Load()))

}
