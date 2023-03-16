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
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(self.ArweaveCurrentHeight, prometheus.CounterValue, float64(self.monitor.Report.ArweaveCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.ArweaveLastNetworkInfoTimestamp, prometheus.CounterValue, float64(self.monitor.Report.ArweaveLastNetworkInfoTimestamp.Load()))
	ch <- prometheus.MustNewConstMetric(self.StartTimestamp, prometheus.CounterValue, float64(self.monitor.Report.StartTimestamp.Load()))
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.CounterValue, float64(self.monitor.Report.UpForSeconds.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerBlocksBehind, prometheus.CounterValue, float64(self.monitor.Report.SyncerBlocksBehind.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerCurrentHeight, prometheus.CounterValue, float64(self.monitor.Report.SyncerCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerFinishedHeight, prometheus.CounterValue, float64(self.monitor.Report.SyncerFinishedHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageBlocksProcessedPerMinute, prometheus.CounterValue, float64(self.monitor.Report.AverageBlocksProcessedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageTransactionDownloadedPerMinute, prometheus.CounterValue, float64(self.monitor.Report.AverageTransactionDownloadedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.AverageInteractionsSavedPerMinute, prometheus.CounterValue, float64(self.monitor.Report.AverageInteractionsSavedPerMinute.Load()))
	ch <- prometheus.MustNewConstMetric(self.PeersBlacklisted, prometheus.CounterValue, float64(self.monitor.Report.PeersBlacklisted.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumPeers, prometheus.CounterValue, float64(self.monitor.Report.NumPeers.Load()))
	ch <- prometheus.MustNewConstMetric(self.TransactionsDownloaded, prometheus.CounterValue, float64(self.monitor.Report.TransactionsDownloaded.Load()))
	ch <- prometheus.MustNewConstMetric(self.InteractionsSaved, prometheus.CounterValue, float64(self.monitor.Report.InteractionsSaved.Load()))
	ch <- prometheus.MustNewConstMetric(self.FailedInteractionParsing, prometheus.CounterValue, float64(self.monitor.Report.FailedInteractionParsing.Load()))
	ch <- prometheus.MustNewConstMetric(self.NumWatchdogRestarts, prometheus.CounterValue, float64(self.monitor.Report.NumWatchdogRestarts.Load()))
}
