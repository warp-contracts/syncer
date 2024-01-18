package monitor_redstone_tx_syncer

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds *prometheus.Desc

	// Errors
	BlockDownloaderFailures               *prometheus.Desc
	SyncerWriteInteractionFailures        *prometheus.Desc
	SyncerWriteInteractionsPermanentError *prometheus.Desc
	SyncerUpdateSyncStateFailures         *prometheus.Desc
	StoreGetLastStateFailure              *prometheus.Desc
	StoreSaveLastStateFailure             *prometheus.Desc

	// State
	BlockDownloaderCurrentHeight *prometheus.Desc
	SyncerTxsProcessed           *prometheus.Desc
	SyncerBlocksProcessed        *prometheus.Desc
	SyncerInteractionsToWarpy    *prometheus.Desc
	StoreLastSyncedBlockHeight   *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		UpForSeconds: prometheus.NewDesc("up_for_seconds", "", nil, nil),

		// Errors
		BlockDownloaderFailures:               prometheus.NewDesc("block_downloader_failures", "", nil, nil),
		SyncerWriteInteractionFailures:        prometheus.NewDesc("syncer_write_interaction_failures", "", nil, nil),
		SyncerWriteInteractionsPermanentError: prometheus.NewDesc("syncer_write_interaction_permanent_error", "", nil, nil),
		StoreGetLastStateFailure:              prometheus.NewDesc("store_get_last_state_failure", "", nil, nil),
		StoreSaveLastStateFailure:             prometheus.NewDesc("store_save_last_state_failure", "", nil, nil),

		// State
		BlockDownloaderCurrentHeight: prometheus.NewDesc("block_downloader_current_height", "", nil, nil),
		SyncerTxsProcessed:           prometheus.NewDesc("syncer_txs_processed", "", nil, nil),
		SyncerBlocksProcessed:        prometheus.NewDesc("syncer_blocks_processed", "", nil, nil),
		SyncerInteractionsToWarpy:    prometheus.NewDesc("syncer_interactions_to_warpy", "", nil, nil),
		StoreLastSyncedBlockHeight:   prometheus.NewDesc("store_last_synced_block_height", "", nil, nil),
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
	ch <- self.SyncerWriteInteractionFailures
	ch <- self.SyncerWriteInteractionsPermanentError
	ch <- self.StoreGetLastStateFailure
	ch <- self.StoreSaveLastStateFailure

	// State
	ch <- self.BlockDownloaderCurrentHeight
	ch <- self.SyncerTxsProcessed
	ch <- self.SyncerBlocksProcessed
	ch <- self.SyncerInteractionsToWarpy
	ch <- self.StoreLastSyncedBlockHeight
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))

	// Errors
	ch <- prometheus.MustNewConstMetric(self.BlockDownloaderFailures, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.BlockDownloaderFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerWriteInteractionFailures, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.SyncerWriteInteractionFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerWriteInteractionsPermanentError, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.SyncerWriteInteractionsPermanentError.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreGetLastStateFailure, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.StoreGetLastStateFailure.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreSaveLastStateFailure, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.Errors.StoreSaveLastStateFailure.Load()))

	// State
	ch <- prometheus.MustNewConstMetric(self.BlockDownloaderCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.RedstoneTxSyncer.State.BlockDownloaderCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerTxsProcessed, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.State.SyncerTxsProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerBlocksProcessed, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.State.SyncerBlocksProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerInteractionsToWarpy, prometheus.CounterValue, float64(self.monitor.Report.RedstoneTxSyncer.State.SyncerInteractionsToWarpy.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreLastSyncedBlockHeight, prometheus.GaugeValue, float64(self.monitor.Report.RedstoneTxSyncer.State.StoreLastSyncedBlockHeight.Load()))
}
