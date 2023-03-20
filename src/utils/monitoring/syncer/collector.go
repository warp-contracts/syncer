package monitor_syncer

import (
	"github.com/prometheus/client_golang/prometheus"
)

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
	labels := prometheus.Labels{
		"app": "syncer",
	}

	return &Collector{
		ArweaveCurrentHeight:                  prometheus.NewDesc("arweave_current_height", "", nil, labels),
		ArweaveLastNetworkInfoTimestamp:       prometheus.NewDesc("arweave_last_network_info_timestamp", "", nil, labels),
		StartTimestamp:                        prometheus.NewDesc("start_timestamp", "", nil, labels),
		UpForSeconds:                          prometheus.NewDesc("up_for_seconds", "", nil, labels),
		SyncerBlocksBehind:                    prometheus.NewDesc("syncer_blocks_behind", "", nil, labels),
		SyncerCurrentHeight:                   prometheus.NewDesc("syncer_current_height", "", nil, labels),
		SyncerFinishedHeight:                  prometheus.NewDesc("syncer_finished_height", "", nil, labels),
		AverageBlocksProcessedPerMinute:       prometheus.NewDesc("average_blocks_processed_per_minute", "", nil, labels),
		AverageTransactionDownloadedPerMinute: prometheus.NewDesc("average_transactions_downloaded_per_minute", "", nil, labels),
		AverageInteractionsSavedPerMinute:     prometheus.NewDesc("average_interactions_saved_per_minute", "", nil, labels),
		PeersBlacklisted:                      prometheus.NewDesc("peers_blacklisted", "", nil, labels),
		NumPeers:                              prometheus.NewDesc("num_peers", "", nil, labels),
		TransactionsDownloaded:                prometheus.NewDesc("transactions_downloaded", "", nil, labels),
		InteractionsSaved:                     prometheus.NewDesc("interactions_saved", "", nil, labels),
		FailedInteractionParsing:              prometheus.NewDesc("failed_interaction_parsing", "", nil, labels),
		NumWatchdogRestarts:                   prometheus.NewDesc("num_watchdog_restarts", "", nil, labels),

		// Errors
		DbInteractionInsert:               prometheus.NewDesc("error_db_interaction", "", nil, labels),
		DbLastTransactionBlockHeightError: prometheus.NewDesc("error_db_last_tx_block_height", "", nil, labels),
		TxValidationErrors:                prometheus.NewDesc("error_tx_validation", "", nil, labels),
		TxDownloadErrors:                  prometheus.NewDesc("error_tx_download", "", nil, labels),
		BlockValidationErrors:             prometheus.NewDesc("error_block_validation", "", nil, labels),
		BlockDownloadErrors:               prometheus.NewDesc("error_block_download", "", nil, labels),
		PeerDownloadErrors:                prometheus.NewDesc("error_peer_download", "", nil, labels),
		NetworkInfoDownloadErrors:         prometheus.NewDesc("error_network_info_download", "", nil, labels),
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
	ch <- prometheus.MustNewConstMetric(self.ArweaveCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.NetworkInfo.State.ArweaveCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.ArweaveLastNetworkInfoTimestamp, prometheus.GaugeValue, float64(self.monitor.Report.NetworkInfo.State.ArweaveLastNetworkInfoTimestamp.Load()))
	ch <- prometheus.MustNewConstMetric(self.StartTimestamp, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.StartTimestamp.Load()))
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.UpForSeconds.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerBlocksBehind, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.SyncerBlocksBehind.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.SyncerCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerFinishedHeight, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.SyncerFinishedHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageBlocksProcessedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.AverageBlocksProcessedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageTransactionDownloadedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.AverageTransactionDownloadedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageInteractionsSavedPerMinute, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.AverageInteractionsSavedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.PeersBlacklisted, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.PeersBlacklisted.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumPeers, prometheus.GaugeValue, float64(self.monitor.Report.Syncer.State.NumPeers.Load()))
	ch <- prometheus.MustNewConstMetric(self.TransactionsDownloaded, prometheus.CounterValue, float64(self.monitor.Report.Syncer.State.TransactionsDownloaded.Load()))
	ch <- prometheus.MustNewConstMetric(self.InteractionsSaved, prometheus.CounterValue, float64(self.monitor.Report.Syncer.State.InteractionsSaved.Load()))
	ch <- prometheus.MustNewConstMetric(self.FailedInteractionParsing, prometheus.CounterValue, float64(self.monitor.Report.Syncer.State.FailedInteractionParsing.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumWatchdogRestarts, prometheus.CounterValue, float64(self.monitor.Report.Syncer.State.NumWatchdogRestarts.Load()))

	// Errors
	ch <- prometheus.MustNewConstMetric(self.NetworkInfoDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.NetworkInfo.Errors.NetworkInfoDownloadErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.DbInteractionInsert, prometheus.CounterValue, float64(self.monitor.Report.Syncer.Errors.DbInteractionInsert.Load()))
	ch <- prometheus.MustNewConstMetric(self.DbLastTransactionBlockHeightError, prometheus.CounterValue, float64(self.monitor.Report.Syncer.Errors.DbLastTransactionBlockHeightError.Load()))
	ch <- prometheus.MustNewConstMetric(self.TxValidationErrors, prometheus.CounterValue, float64(self.monitor.Report.Syncer.Errors.TxValidationErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.TxDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Syncer.Errors.TxDownloadErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlockValidationErrors, prometheus.CounterValue, float64(self.monitor.Report.Syncer.Errors.BlockValidationErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.BlockDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Syncer.Errors.BlockDownloadErrors.Load()))
	ch <- prometheus.MustNewConstMetric(self.PeerDownloadErrors, prometheus.CounterValue, float64(self.monitor.Report.Syncer.Errors.PeerDownloadErrors.Load()))
}
