package monitor_warpy_syncer

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds *prometheus.Desc

	// Errors
	BlockDownloaderFailures                *prometheus.Desc
	SyncerDeltaCheckTxFailures             *prometheus.Desc
	SyncerDeltaProcessTxPermanentError     *prometheus.Desc
	SyncerSommelierProcessTxPermanentError *prometheus.Desc
	SyncerSommelierCheckTxFailures         *prometheus.Desc
	SyncerDeltaUpdateSyncStateFailures     *prometheus.Desc
	WriterFailures                         *prometheus.Desc
	StoreGetLastStateFailure               *prometheus.Desc
	StoreSaveLastStateFailure              *prometheus.Desc
	PollerSommelierFetchError              *prometheus.Desc
	StoreSommelierFailures                 *prometheus.Desc

	// State
	BlockDownloaderCurrentHeight     *prometheus.Desc
	SyncerDeltaTxsProcessed          *prometheus.Desc
	SyncerDeltaBlocksProcessed       *prometheus.Desc
	SyncerSommelierTxsProcessed      *prometheus.Desc
	SyncerSommelierBlocksProcessed   *prometheus.Desc
	WriterInteractionsToWarpy        *prometheus.Desc
	StoreLastSyncedBlockHeight       *prometheus.Desc
	PollerSommelierAssetsFromSelects *prometheus.Desc
	StoreSommelierRecordsSaved       *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		UpForSeconds: prometheus.NewDesc("up_for_seconds", "", nil, nil),

		// Errors
		BlockDownloaderFailures:                prometheus.NewDesc("block_downloader_failures", "", nil, nil),
		SyncerDeltaCheckTxFailures:             prometheus.NewDesc("syncer_delta_check_tx_failures", "", nil, nil),
		SyncerSommelierCheckTxFailures:         prometheus.NewDesc("syncer_sommelier_check_tx_failures", "", nil, nil),
		SyncerDeltaProcessTxPermanentError:     prometheus.NewDesc("syncer_delta_process_tx_permanent_error", "", nil, nil),
		SyncerSommelierProcessTxPermanentError: prometheus.NewDesc("syncer_sommelier_process_tx_permanent_error", "", nil, nil),
		WriterFailures:                         prometheus.NewDesc("writer_failures", "", nil, nil),
		StoreGetLastStateFailure:               prometheus.NewDesc("store_get_last_state_failure", "", nil, nil),
		StoreSaveLastStateFailure:              prometheus.NewDesc("store_save_last_state_failure", "", nil, nil),
		PollerSommelierFetchError:              prometheus.NewDesc("poller_sommelier_fetch_error", "", nil, nil),
		StoreSommelierFailures:                 prometheus.NewDesc("store_sommelier_failures", "", nil, nil),

		// State
		BlockDownloaderCurrentHeight:     prometheus.NewDesc("block_downloader_current_height", "", nil, nil),
		SyncerDeltaTxsProcessed:          prometheus.NewDesc("syncer_delta_txs_processed", "", nil, nil),
		SyncerDeltaBlocksProcessed:       prometheus.NewDesc("syncer_delta_blocks_processed", "", nil, nil),
		SyncerSommelierTxsProcessed:      prometheus.NewDesc("syncer_sommelier_txs_processed", "", nil, nil),
		SyncerSommelierBlocksProcessed:   prometheus.NewDesc("syncer_sommelier_blocks_processed", "", nil, nil),
		WriterInteractionsToWarpy:        prometheus.NewDesc("writer_interactions_to_warpy", "", nil, nil),
		StoreLastSyncedBlockHeight:       prometheus.NewDesc("store_last_synced_block_height", "", nil, nil),
		PollerSommelierAssetsFromSelects: prometheus.NewDesc("poller_sommelier_assets_from_selects", "", nil, nil),
		StoreSommelierRecordsSaved:       prometheus.NewDesc("store_sommelier_records_saved", "", nil, nil),
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
	ch <- self.SyncerDeltaCheckTxFailures
	ch <- self.SyncerDeltaProcessTxPermanentError
	ch <- self.SyncerSommelierProcessTxPermanentError
	ch <- self.SyncerSommelierCheckTxFailures
	ch <- self.WriterFailures
	ch <- self.StoreGetLastStateFailure
	ch <- self.StoreSaveLastStateFailure
	ch <- self.PollerSommelierFetchError
	ch <- self.StoreSommelierFailures

	// State
	ch <- self.BlockDownloaderCurrentHeight
	ch <- self.SyncerDeltaTxsProcessed
	ch <- self.SyncerDeltaBlocksProcessed
	ch <- self.WriterInteractionsToWarpy
	ch <- self.StoreLastSyncedBlockHeight
	ch <- self.PollerSommelierAssetsFromSelects
	ch <- self.StoreSommelierRecordsSaved
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))

	// Errors
	ch <- prometheus.MustNewConstMetric(self.BlockDownloaderFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.BlockDownloaderFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDeltaCheckTxFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.SyncerDeltaCheckTxFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.WriterFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.WriterFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDeltaProcessTxPermanentError, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.SyncerDeltaProcessTxPermanentError.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerSommelierCheckTxFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.SyncerSommelierCheckTxFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerSommelierProcessTxPermanentError, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.SyncerSommelierProcessTxPermanentError.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreGetLastStateFailure, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.StoreGetLastStateFailure.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreSaveLastStateFailure, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.StoreSaveLastStateFailure.Load()))
	ch <- prometheus.MustNewConstMetric(self.PollerSommelierFetchError, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.PollerSommelierFetchError.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreSommelierFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.StoreSommelierFailures.Load()))

	// State
	ch <- prometheus.MustNewConstMetric(self.BlockDownloaderCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.WarpySyncer.State.BlockDownloaderCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDeltaTxsProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerDeltaTxsProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDeltaBlocksProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerDeltaBlocksProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerSommelierTxsProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerSommelierTxsProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerSommelierBlocksProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerSommelierBlocksProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.WriterInteractionsToWarpy, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.WriterInteractionsToWarpy.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreLastSyncedBlockHeight, prometheus.GaugeValue, float64(self.monitor.Report.WarpySyncer.State.StoreLastSyncedBlockHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.PollerSommelierAssetsFromSelects, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.PollerSommelierAssetsFromSelects.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreSommelierRecordsSaved, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.StoreSommelierRecordsSaved.Load()))
}
