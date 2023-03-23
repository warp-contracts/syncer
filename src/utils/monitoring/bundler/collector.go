package monitor_bundler

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	BundlesFromNotifications    *prometheus.Desc
	AdditionalFetches           *prometheus.Desc
	BundlesFromSelects          *prometheus.Desc
	AllBundlesFromDb            *prometheus.Desc
	BundlrSuccess               *prometheus.Desc
	ConfirmationsSavedToDb      *prometheus.Desc
	BundrlError                 *prometheus.Desc
	ConfirmationsSavedToDbError *prometheus.Desc
	AdditionalFetchError        *prometheus.Desc
}

func NewCollector() *Collector {
	labels := prometheus.Labels{
		"app": "bundler",
	}

	return &Collector{
		BundlesFromNotifications:    prometheus.NewDesc("bundles_from_notifications", "", nil, labels),
		AdditionalFetches:           prometheus.NewDesc("additional_fetches", "", nil, labels),
		BundlesFromSelects:          prometheus.NewDesc("bundles_from_selects", "", nil, labels),
		AllBundlesFromDb:            prometheus.NewDesc("all_bundles_from_db", "", nil, labels),
		BundlrSuccess:               prometheus.NewDesc("bundlr_success", "", nil, labels),
		ConfirmationsSavedToDb:      prometheus.NewDesc("confirmations_saved_to_db", "", nil, labels),
		BundrlError:                 prometheus.NewDesc("bundrl_error", "", nil, labels),
		ConfirmationsSavedToDbError: prometheus.NewDesc("confirmations_saved_to_db_error", "", nil, labels),
		AdditionalFetchError:        prometheus.NewDesc("additional_fetch_error", "", nil, labels),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- self.BundlesFromNotifications
	ch <- self.BundlesFromSelects
	ch <- self.AdditionalFetches
	ch <- self.AllBundlesFromDb
	ch <- self.BundlrSuccess
	ch <- self.ConfirmationsSavedToDb

	// Errors
	ch <- self.BundrlError
	ch <- self.ConfirmationsSavedToDbError
	ch <- self.AdditionalFetchError
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(self.BundlesFromNotifications, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.BundlesFromNotifications.Load()))
	ch <- prometheus.MustNewConstMetric(self.AdditionalFetches, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.AdditionalFetches.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundlesFromSelects, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.BundlesFromSelects.Load()))
	ch <- prometheus.MustNewConstMetric(self.AllBundlesFromDb, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.AllBundlesFromDb.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundlrSuccess, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.BundlrSuccess.Load()))
	ch <- prometheus.MustNewConstMetric(self.ConfirmationsSavedToDb, prometheus.CounterValue, float64(self.monitor.Report.Bundler.State.ConfirmationsSavedToDb.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundrlError, prometheus.CounterValue, float64(self.monitor.Report.Bundler.Errors.BundrlError.Load()))
	ch <- prometheus.MustNewConstMetric(self.ConfirmationsSavedToDbError, prometheus.CounterValue, float64(self.monitor.Report.Bundler.Errors.ConfirmationsSavedToDbError.Load()))
	ch <- prometheus.MustNewConstMetric(self.AdditionalFetchError, prometheus.CounterValue, float64(self.monitor.Report.Bundler.Errors.AdditionalFetchError.Load()))
}
