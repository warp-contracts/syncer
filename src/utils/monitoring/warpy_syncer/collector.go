package monitor_warpy_syncer

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds *prometheus.Desc

	// Errors
	BlockDownloaderFailures              *prometheus.Desc
	SyncerDeltaCheckTxFailures           *prometheus.Desc
	SyncerDeltaProcessTxPermanentError   *prometheus.Desc
	SyncerDepositProcessTxPermanentError *prometheus.Desc
	SyncerDepositCheckTxFailures         *prometheus.Desc
	SyncerDeltaUpdateSyncStateFailures   *prometheus.Desc
	WriterFailures                       *prometheus.Desc
	StoreGetLastStateFailure             *prometheus.Desc
	StoreSaveLastStateFailure            *prometheus.Desc
	PollerDepositFetchError              *prometheus.Desc
	StoreDepositFailures                 *prometheus.Desc
	AssetsCalculatorFailures             *prometheus.Desc

	// State
	BlockDownloaderCurrentHeight   *prometheus.Desc
	SyncerDeltaTxsProcessed        *prometheus.Desc
	SyncerDeltaBlocksProcessed     *prometheus.Desc
	SyncerDepositTxsProcessed      *prometheus.Desc
	SyncerDepositBlocksProcessed   *prometheus.Desc
	WriterInteractionsToWarpy      *prometheus.Desc
	StoreLastSyncedBlockHeight     *prometheus.Desc
	PollerDepositAssetsFromSelects *prometheus.Desc
	StoreDepositRecordsSaved       *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		UpForSeconds: prometheus.NewDesc("up_for_seconds", "", nil, nil),

		// Errors
		BlockDownloaderFailures:              prometheus.NewDesc("block_downloader_failures", "", nil, nil),
		SyncerDeltaCheckTxFailures:           prometheus.NewDesc("syncer_delta_check_tx_failures", "", nil, nil),
		SyncerDepositCheckTxFailures:         prometheus.NewDesc("syncer_deposit_check_tx_failures", "", nil, nil),
		SyncerDeltaProcessTxPermanentError:   prometheus.NewDesc("syncer_delta_process_tx_permanent_error", "", nil, nil),
		SyncerDepositProcessTxPermanentError: prometheus.NewDesc("syncer_deposit_process_tx_permanent_error", "", nil, nil),
		WriterFailures:                       prometheus.NewDesc("writer_failures", "", nil, nil),
		StoreGetLastStateFailure:             prometheus.NewDesc("store_get_last_state_failure", "", nil, nil),
		StoreSaveLastStateFailure:            prometheus.NewDesc("store_save_last_state_failure", "", nil, nil),
		PollerDepositFetchError:              prometheus.NewDesc("poller_deposit_fetch_error", "", nil, nil),
		StoreDepositFailures:                 prometheus.NewDesc("store_deposit_failures", "", nil, nil),
		AssetsCalculatorFailures:             prometheus.NewDesc("assets_calculator_failures", "", nil, nil),

		// State
		BlockDownloaderCurrentHeight:   prometheus.NewDesc("block_downloader_current_height", "", nil, nil),
		SyncerDeltaTxsProcessed:        prometheus.NewDesc("syncer_delta_txs_processed", "", nil, nil),
		SyncerDeltaBlocksProcessed:     prometheus.NewDesc("syncer_delta_blocks_processed", "", nil, nil),
		SyncerDepositTxsProcessed:      prometheus.NewDesc("syncer_deposit_txs_processed", "", nil, nil),
		SyncerDepositBlocksProcessed:   prometheus.NewDesc("syncer_deposit_blocks_processed", "", nil, nil),
		WriterInteractionsToWarpy:      prometheus.NewDesc("writer_interactions_to_warpy", "", nil, nil),
		StoreLastSyncedBlockHeight:     prometheus.NewDesc("store_last_synced_block_height", "", nil, nil),
		PollerDepositAssetsFromSelects: prometheus.NewDesc("poller_deposit_assets_from_selects", "", nil, nil),
		StoreDepositRecordsSaved:       prometheus.NewDesc("store_deposit_records_saved", "", nil, nil),
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
	ch <- self.SyncerDepositProcessTxPermanentError
	ch <- self.SyncerDepositCheckTxFailures
	ch <- self.WriterFailures
	ch <- self.StoreGetLastStateFailure
	ch <- self.StoreSaveLastStateFailure
	ch <- self.PollerDepositFetchError
	ch <- self.StoreDepositFailures
	ch <- self.AssetsCalculatorFailures

	// State
	ch <- self.BlockDownloaderCurrentHeight
	ch <- self.SyncerDeltaTxsProcessed
	ch <- self.SyncerDeltaBlocksProcessed
	ch <- self.WriterInteractionsToWarpy
	ch <- self.StoreLastSyncedBlockHeight
	ch <- self.PollerDepositAssetsFromSelects
	ch <- self.StoreDepositRecordsSaved
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
	ch <- prometheus.MustNewConstMetric(self.SyncerDepositCheckTxFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.SyncerDepositCheckTxFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDepositProcessTxPermanentError, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.SyncerDepositProcessTxPermanentError.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreGetLastStateFailure, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.StoreGetLastStateFailure.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreSaveLastStateFailure, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.StoreSaveLastStateFailure.Load()))
	ch <- prometheus.MustNewConstMetric(self.PollerDepositFetchError, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.PollerDepositFetchError.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreDepositFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.StoreDepositFailures.Load()))
	ch <- prometheus.MustNewConstMetric(self.AssetsCalculatorFailures, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.Errors.AssetsCalculatorFailures.Load()))

	// State
	ch <- prometheus.MustNewConstMetric(self.BlockDownloaderCurrentHeight, prometheus.GaugeValue, float64(self.monitor.Report.WarpySyncer.State.BlockDownloaderCurrentHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDeltaTxsProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerDeltaTxsProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDeltaBlocksProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerDeltaBlocksProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDepositTxsProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerDepositTxsProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.SyncerDepositBlocksProcessed, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.SyncerDepositBlocksProcessed.Load()))
	ch <- prometheus.MustNewConstMetric(self.WriterInteractionsToWarpy, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.WriterInteractionsToWarpy.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreLastSyncedBlockHeight, prometheus.GaugeValue, float64(self.monitor.Report.WarpySyncer.State.StoreLastSyncedBlockHeight.Load()))
	ch <- prometheus.MustNewConstMetric(self.PollerDepositAssetsFromSelects, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.PollerDepositAssetsFromSelects.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreDepositRecordsSaved, prometheus.CounterValue, float64(self.monitor.Report.WarpySyncer.State.StoreDepositRecordsSaved.Load()))
}
