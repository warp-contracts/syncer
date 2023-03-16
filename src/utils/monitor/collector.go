package monitor

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Define a struct for you collector that contains pointers
// to prometheus descriptors for each metric you wish to expose.
// Note you can also include fields of other types if they provide utility
// but we just won't be exposing them as metrics.
type Collector struct {
	monitor *Monitor

	ArweaveCurrentHeight                  *prometheus.Desc
	ArweaveLastNetworkInfoTimestamp       *prometheus.Desc
	StartTimestamp                        *prometheus.Desc
	UpForSeconds                          *prometheus.Desc
	SyncerBlocksBehind                    *prometheus.Desc
	SyncerCurrentHeight                   *prometheus.Desc
	SyncerFinishedHeight                  *prometheus.Desc
	AverageBlocksProcessedPerMinute       *prometheus.Desc
	AverageTransactionDownloadedPerMinute *prometheus.Desc
	AverageInteractionsSavedPerMinute     *prometheus.Desc
	PeersBlacklisted                      *prometheus.Desc
	NumPeers                              *prometheus.Desc
	TransactionsDownloaded                *prometheus.Desc
	InteractionsSaved                     *prometheus.Desc
	FailedInteractionParsing              *prometheus.Desc
	NumWatchdogRestarts                   *prometheus.Desc

	DbInteractionInsert               *prometheus.Desc `json:""`
	DbLastTransactionBlockHeightError *prometheus.Desc `json:""`
	TxValidationErrors                *prometheus.Desc `json:""`
	TxDownloadErrors                  *prometheus.Desc `json:""`
	BlockValidationErrors             *prometheus.Desc `json:""`
	BlockDownloadErrors               *prometheus.Desc `json:""`
	PeerDownloadErrors                *prometheus.Desc `json:""`
	NetworkInfoDownloadErrors         *prometheus.Desc `json:""`
}

func NewCollector() *Collector {
	return &Collector{
		ArweaveCurrentHeight:                  prometheus.NewDesc("arweave_current_height", "", nil, nil),
		ArweaveLastNetworkInfoTimestamp:       prometheus.NewDesc("arweave_last_network_info_timestamp", "", nil, nil),
		StartTimestamp:                        prometheus.NewDesc("start_timestamp", "", nil, nil),
		UpForSeconds:                          prometheus.NewDesc("up_for_seconds", "", nil, nil),
		SyncerBlocksBehind:                    prometheus.NewDesc("syncer_blocks_behind", "", nil, nil),
		SyncerCurrentHeight:                   prometheus.NewDesc("syncer_current_height", "", nil, nil),
		SyncerFinishedHeight:                  prometheus.NewDesc("syncer_finished_height", "", nil, nil),
		AverageBlocksProcessedPerMinute:       prometheus.NewDesc("average_blocks_processed_per_minute", "", nil, nil),
		AverageTransactionDownloadedPerMinute: prometheus.NewDesc("average_transactions_downloaded_per_minute", "", nil, nil),
		AverageInteractionsSavedPerMinute:     prometheus.NewDesc("average_interactions_saved_per_minute", "", nil, nil),
		PeersBlacklisted:                      prometheus.NewDesc("peers_blacklisted", "", nil, nil),
		NumPeers:                              prometheus.NewDesc("num_peers", "", nil, nil),
		TransactionsDownloaded:                prometheus.NewDesc("transactions_downloaded", "", nil, nil),
		InteractionsSaved:                     prometheus.NewDesc("interactions_saved", "", nil, nil),
		FailedInteractionParsing:              prometheus.NewDesc("failed_interaction_parsing", "", nil, nil),
		NumWatchdogRestarts:                   prometheus.NewDesc("num_watchdog_restarts", "", nil, nil),

		// Errors
		DbInteractionInsert:               prometheus.NewDesc("error_db_interaction", "", nil, nil),
		DbLastTransactionBlockHeightError: prometheus.NewDesc("error_db_last_tx_block_height", "", nil, nil),
		TxValidationErrors:                prometheus.NewDesc("error_tx_validation", "", nil, nil),
		TxDownloadErrors:                  prometheus.NewDesc("error_tx_download", "", nil, nil),
		BlockValidationErrors:             prometheus.NewDesc("error_block_validation", "", nil, nil),
		BlockDownloadErrors:               prometheus.NewDesc("error_block_download", "", nil, nil),
		PeerDownloadErrors:                prometheus.NewDesc("error_peer_download", "", nil, nil),
		NetworkInfoDownloadErrors:         prometheus.NewDesc("error_network_info_download", "", nil, nil),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- self.ArweaveCurrentHeight
	ch <- self.ArweaveLastNetworkInfoTimestamp
	ch <- self.StartTimestamp
	ch <- self.UpForSeconds
	ch <- self.SyncerBlocksBehind
	ch <- self.SyncerCurrentHeight
	ch <- self.SyncerFinishedHeight
	ch <- self.AverageBlocksProcessedPerMinute
	ch <- self.AverageTransactionDownloadedPerMinute
	ch <- self.AverageInteractionsSavedPerMinute
	ch <- self.PeersBlacklisted
	ch <- self.NumPeers
	ch <- self.TransactionsDownloaded
	ch <- self.InteractionsSaved
	ch <- self.FailedInteractionParsing
	ch <- self.NumWatchdogRestarts

	// Errors
	ch <- self.DbInteractionInsert
	ch <- self.DbLastTransactionBlockHeightError
	ch <- self.TxValidationErrors
	ch <- self.TxDownloadErrors
	ch <- self.BlockValidationErrors
	ch <- self.BlockDownloadErrors
	ch <- self.PeerDownloadErrors
	ch <- self.NetworkInfoDownloadErrors
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(self.ArweaveCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.ArweaveCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.ArweaveLastNetworkInfoTimestamp, prometheus.GaugeValue, float64(self.monitor.Report.ArweaveLastNetworkInfoTimestamp.Load()))
	ch <- prometheus.MustNewConstMetric(self.StartTimestamp, prometheus.GaugeValue, float64(self.monitor.Report.StartTimestamp.Load()))
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.UpForSeconds.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerBlocksBehind, prometheus.GaugeValue, float64(self.monitor.Report.SyncerBlocksBehind.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.SyncerCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerFinishedHeight, prometheus.GaugeValue, float64(self.monitor.Report.SyncerFinishedHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageBlocksProcessedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.AverageBlocksProcessedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageTransactionDownloadedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.AverageTransactionDownloadedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageInteractionsSavedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.AverageInteractionsSavedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.PeersBlacklisted, prometheus.GaugeValue, float64(self.monitor.Report.PeersBlacklisted.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumPeers, prometheus.GaugeValue, float64(self.monitor.Report.NumPeers.Load()))
	ch <- prometheus.MustNewConstMetric(self.TransactionsDownloaded, prometheus.CounterValue, float64(self.monitor.Report.TransactionsDownloaded.Load()))
	ch <- prometheus.MustNewConstMetric(self.InteractionsSaved, prometheus.CounterValue, float64(self.monitor.Report.InteractionsSaved.Load()))
	ch <- prometheus.MustNewConstMetric(self.FailedInteractionParsing, prometheus.CounterValue, float64(self.monitor.Report.FailedInteractionParsing.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumWatchdogRestarts, prometheus.CounterValue, float64(self.monitor.Report.NumWatchdogRestarts.Load()))

	// Errors
	ch <- prometheus.MustNewConstMetric(self.DbInteractionInsert, prometheus.CounterValue, float64(self.monitor.Report.Errors.DbInteractionInsert.Load()))
	ch <- prometheus.MustNewConstMetric(self.DbLastTransactionBlockHeightError, prometheus.CounterValue, float64(self.monitor.Report.Errors.DbLastTransactionBlockHeightError.Load()))
	ch <- prometheus.MustNewConstMetric(self.TxValidationErrors, prometheus.CounterValue, float64(self.monitor.Report.Errors.TxValidationErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.TxDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Errors.TxDownloadErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlockValidationErrors, prometheus.CounterValue, float64(self.monitor.Report.Errors.BlockValidationErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlockDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Errors.BlockDownloadErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.PeerDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Errors.PeerDownloadErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.NetworkInfoDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Errors.NetworkInfoDownloadErrors.Load()))
}
