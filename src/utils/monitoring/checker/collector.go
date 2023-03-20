package monitor_checker

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Define a struct for you collector that contains pointers
// to prometheus descriptors for each metric you wish to expose.
// Note you can also include fields of other types if they provide utility
// but we just won't be exposing them as metrics.
type Collector struct {
	monitor *Monitor

	BundlesFromNotifications    *prometheus.Desc `json:"bundles_from_notifications"`
	BundlesFromSelects          *prometheus.Desc `json:"bundles_from_selects"`
	AllBundlesFromDb            *prometheus.Desc `json:"all_bundles_from_db"`
	BundlrSuccess               *prometheus.Desc `json:"bundlr_success"`
	ConfirmationsSavedToDb      *prometheus.Desc `json:"confirmations_saved_to_db"`
	BundrlError                 *prometheus.Desc `json:"bundrl_error"`
	ConfirmationsSavedToDbError *prometheus.Desc `json:"confirmations_saved_to_db_error"`
}

func NewCollector() *Collector {
	labels := prometheus.Labels{
		"app": "bundler",
	}

	return &Collector{
		BundlesFromNotifications:    prometheus.NewDesc("bundles_from_notifications", "", nil, labels),
		BundlesFromSelects:          prometheus.NewDesc("bundles_from_selects", "", nil, labels),
		AllBundlesFromDb:            prometheus.NewDesc("all_bundles_from_db", "", nil, labels),
		BundlrSuccess:               prometheus.NewDesc("bundlr_success", "", nil, labels),
		ConfirmationsSavedToDb:      prometheus.NewDesc("confirmations_saved_to_db", "", nil, labels),
		BundrlError:                 prometheus.NewDesc("bundrl_error", "", nil, labels),
		ConfirmationsSavedToDbError: prometheus.NewDesc("confirmations_saved_to_db_error", "", nil, labels),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- self.BundlesFromNotifications
	ch <- self.BundlesFromSelects
	ch <- self.AllBundlesFromDb
	ch <- self.BundlrSuccess
	ch <- self.ConfirmationsSavedToDb

	// Errors
	ch <- self.BundrlError
	ch <- self.ConfirmationsSavedToDbError
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(self.BundlesFromNotifications, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.BundlesFromNotifications.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundlesFromSelects, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.BundlesFromSelects.Load()))
	ch <- prometheus.MustNewConstMetric(self.AllBundlesFromDb, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.AllBundlesFromDb.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundlrSuccess, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.BundlrSuccess.Load()))
	ch <- prometheus.MustNewConstMetric(self.ConfirmationsSavedToDb, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.ConfirmationsSavedToDb.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundrlError, prometheus.CounterValue, float64(self.monitor.Report.Bundler.Errors.BundrlError.Load()))
	ch <- prometheus.MustNewConstMetric(self.ConfirmationsSavedToDbError, prometheus.CounterValue, float64(self.monitor.Report.Bundler.Errors.ConfirmationsSavedToDbError.Load()))
}
