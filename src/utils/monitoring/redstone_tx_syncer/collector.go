package monitor_redstone_tx_syncer

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds *prometheus.Desc

	// Errors
	BlockDownloaderFailures         *prometheus.Desc
	SyncerWriteInteractionsFailures *prometheus.Desc
	SyncerUpdateSyncStateFailures   *prometheus.Desc

	// State
	BlockDownloaderCurrentHeight *prometheus.Desc
	SyncerTxsProcessed           *prometheus.Desc
	SyncerBlocksProcessed        *prometheus.Desc
	SyncerInteractionsToWarpy    *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		UpForSeconds: prometheus.NewDesc("up_for_seconds", "", nil, nil),

		// Errors
		BlockDownloaderFailures:         prometheus.NewDesc("block_downloader_failures", "", nil, nil),
		SyncerWriteInteractionsFailures: prometheus.NewDesc("syncer_write_interactions_failures", "", nil, nil),
		SyncerUpdateSyncStateFailures:   prometheus.NewDesc("syncer_update_sync_state_failures", "", nil, nil),

		// State
		BlockDownloaderCurrentHeight: prometheus.NewDesc("block_downloader_current_height", "", nil, nil),
		SyncerTxsProcessed:           prometheus.NewDesc("syncer_txs_processed", "", nil, nil),
		SyncerBlocksProcessed:        prometheus.NewDesc("syncer_blocks_processed", "", nil, nil),
		SyncerInteractionsToWarpy:    prometheus.NewDesc("syncer_interactions_to_warpy", "", nil, nil),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Run
	ch <- self.UpForSeconds

	// Errors
	ch <- self.BlockDownloaderFailures
	ch <- self.SyncerWriteInteractionsFailures
	ch <- self.SyncerUpdateSyncStateFailures

	// State
	ch <- self.BlockDownloaderCurrentHeight
	ch <- self.SyncerTxsProcessed
	ch <- self.SyncerBlocksProcessed
	ch <- self.SyncerInteractionsToWarpy
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))

	// Errors
	ch <- prometheus.MustNewConstMetric(self.BlockDownloaderFailures, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.BlockDownloaderFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerWriteInteractionsFailures, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.SyncerWriteInteractionsFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerUpdateSyncStateFailures, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.SyncerUpdateSyncStateFailures.Load()))

	// State
	ch <- prometheus.MustNewConstMetric(self.BlockDownloaderCurrentHeight, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.State.BlockDownloaderCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerTxsProcessed, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.State.SyncerTxsProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerBlocksProcessed, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.State.SyncerBlocksProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerInteractionsToWarpy, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.State.SyncerInteractionsToWarpy.Load()))
}
