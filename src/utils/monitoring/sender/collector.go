package monitor_sender

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds *prometheus.Desc

	PendingBundleItems          *prometheus.Desc
	BundlesFromNotifications    *prometheus.Desc
	AdditionalFetches           *prometheus.Desc
	BundlesFromSelects          *prometheus.Desc
	RetriedBundlesFromSelects   *prometheus.Desc
	AllBundlesFromDb            *prometheus.Desc
	IrysSuccess                 *prometheus.Desc
	ConfirmationsSavedToDb      *prometheus.Desc
	IrysError                   *prometheus.Desc
	IrysMarshalError            *prometheus.Desc
	ConfirmationsSavedToDbError *prometheus.Desc
	AdditionalFetchError        *prometheus.Desc
	PollerFetchError            *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		UpForSeconds:                prometheus.NewDesc("up_for_seconds", "", nil, nil),
		PendingBundleItems:          prometheus.NewDesc("pending_bundle_items", "", nil, nil),
		BundlesFromNotifications:    prometheus.NewDesc("bundles_from_notifications", "", nil, nil),
		AdditionalFetches:           prometheus.NewDesc("additional_fetches", "", nil, nil),
		BundlesFromSelects:          prometheus.NewDesc("bundles_from_selects", "", nil, nil),
		RetriedBundlesFromSelects:   prometheus.NewDesc("retried_bundles_from_selects", "", nil, nil),
		AllBundlesFromDb:            prometheus.NewDesc("all_bundles_from_db", "", nil, nil),
		IrysSuccess:                 prometheus.NewDesc("irys_success", "", nil, nil),
		ConfirmationsSavedToDb:      prometheus.NewDesc("confirmations_saved_to_db", "", nil, nil),
		IrysError:                   prometheus.NewDesc("irys_error", "", nil, nil),
		IrysMarshalError:            prometheus.NewDesc("irys_marshal_error", "", nil, nil),
		ConfirmationsSavedToDbError: prometheus.NewDesc("confirmations_saved_to_db_error", "", nil, nil),
		AdditionalFetchError:        prometheus.NewDesc("additional_fetch_error", "", nil, nil),
		PollerFetchError:            prometheus.NewDesc("poller_fetch_error", "", nil, nil),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Run
	ch <- self.UpForSeconds

	ch <- self.PendingBundleItems
	ch <- self.BundlesFromNotifications
	ch <- self.BundlesFromSelects
	ch <- self.RetriedBundlesFromSelects
	ch <- self.AdditionalFetches
	ch <- self.AllBundlesFromDb
	ch <- self.IrysSuccess
	ch <- self.ConfirmationsSavedToDb

	// Errors
	ch <- self.IrysError
	ch <- self.ConfirmationsSavedToDbError
	ch <- self.AdditionalFetchError
	ch <- self.PollerFetchError
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))

	// Bundler
	ch <- prometheus.MustNewConstMetric(self.PendingBundleItems, prometheus.GaugeValue, float64(self.monitor.Report.Sender.State.PendingBundleItems.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundlesFromNotifications, prometheus.CounterValue, float64(self.monitor.Report.Sender.State.BundlesFromNotifications.Load()))
	ch <- prometheus.MustNewConstMetric(self.AdditionalFetches, prometheus.CounterValue, float64(self.monitor.Report.Sender.State.AdditionalFetches.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundlesFromSelects, prometheus.CounterValue, float64(self.monitor.Report.Sender.State.BundlesFromSelects.Load()))
	ch <- prometheus.MustNewConstMetric(self.RetriedBundlesFromSelects, prometheus.CounterValue, float64(self.monitor.Report.Sender.State.RetriedBundlesFromSelects.Load()))
	ch <- prometheus.MustNewConstMetric(self.AllBundlesFromDb, prometheus.CounterValue, float64(self.monitor.Report.Sender.State.AllBundlesFromDb.Load()))
	ch <- prometheus.MustNewConstMetric(self.IrysSuccess, prometheus.CounterValue, float64(self.monitor.Report.Sender.State.IrysSuccess.Load()))
	ch <- prometheus.MustNewConstMetric(self.ConfirmationsSavedToDb, prometheus.CounterValue, float64(self.monitor.Report.Sender.State.ConfirmationsSavedToDb.Load()))
	ch <- prometheus.MustNewConstMetric(self.IrysError, prometheus.CounterValue, float64(self.monitor.Report.Sender.Errors.IrysError.Load()))
	ch <- prometheus.MustNewConstMetric(self.IrysMarshalError, prometheus.CounterValue, float64(self.monitor.Report.Sender.Errors.IrysMarshalError.Load()))
	ch <- prometheus.MustNewConstMetric(self.ConfirmationsSavedToDbError, prometheus.CounterValue, float64(self.monitor.Report.Sender.Errors.ConfirmationsSavedToDbError.Load()))
	ch <- prometheus.MustNewConstMetric(self.AdditionalFetchError, prometheus.CounterValue, float64(self.monitor.Report.Sender.Errors.AdditionalFetchError.Load()))
	ch <- prometheus.MustNewConstMetric(self.PollerFetchError, prometheus.CounterValue, float64(self.monitor.Report.Sender.Errors.PollerFetchError.Load()))
}
