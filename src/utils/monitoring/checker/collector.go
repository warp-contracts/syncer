package monitor_checker

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	BundlesTakenFromDb   *prometheus.Desc `json:"bundles_taken_from_db"`
	AllCheckedBundles    *prometheus.Desc `json:"all_checked_bundles"`
	FinishedBundles      *prometheus.Desc `json:"finished_bundles"`
	UnfinishedBundles    *prometheus.Desc `json:"unfinished_bundles"`
	DbStateUpdated       *prometheus.Desc `json:"db_state_updated"`
	BundrlGetStatusError *prometheus.Desc `json:"bundle_check_state_error"`
	DbStateUpdateError   *prometheus.Desc `json:"db_state_update_error"`
}

func NewCollector() *Collector {
	labels := prometheus.Labels{
		"app": "checker",
	}

	return &Collector{
		BundlesTakenFromDb:   prometheus.NewDesc("bundles_taken_from_db", "", nil, labels),
		AllCheckedBundles:    prometheus.NewDesc("all_checked_bundles", "", nil, labels),
		FinishedBundles:      prometheus.NewDesc("finished_bundles", "", nil, labels),
		UnfinishedBundles:    prometheus.NewDesc("unfinished_bundles", "", nil, labels),
		DbStateUpdated:       prometheus.NewDesc("db_state_updated", "", nil, labels),
		BundrlGetStatusError: prometheus.NewDesc("bundle_check_state_error", "", nil, labels),
		DbStateUpdateError:   prometheus.NewDesc("db_state_update_error", "", nil, labels),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- self.BundlesTakenFromDb
	ch <- self.AllCheckedBundles
	ch <- self.FinishedBundles
	ch <- self.UnfinishedBundles
	ch <- self.DbStateUpdated
	ch <- self.BundrlGetStatusError
	ch <- self.DbStateUpdateError
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(self.BundlesTakenFromDb, prometheus.CounterValue, float64(self.monitor.Report.Checker.State.BundlesTakenFromDb.Load()))
	ch <- prometheus.MustNewConstMetric(self.AllCheckedBundles, prometheus.CounterValue, float64(self.monitor.Report.Checker.State.AllCheckedBundles.Load()))
	ch <- prometheus.MustNewConstMetric(self.FinishedBundles, prometheus.CounterValue, float64(self.monitor.Report.Checker.State.FinishedBundles.Load()))
	ch <- prometheus.MustNewConstMetric(self.UnfinishedBundles, prometheus.CounterValue, float64(self.monitor.Report.Checker.State.UnfinishedBundles.Load()))
	ch <- prometheus.MustNewConstMetric(self.DbStateUpdated, prometheus.CounterValue, float64(self.monitor.Report.Checker.State.DbStateUpdated.Load()))
	ch <- prometheus.MustNewConstMetric(self.BundrlGetStatusError, prometheus.CounterValue, float64(self.monitor.Report.Checker.Errors.BundrlGetStatusError.Load()))
	ch <- prometheus.MustNewConstMetric(self.DbStateUpdateError, prometheus.CounterValue, float64(self.monitor.Report.Checker.Errors.DbStateUpdateError.Load()))
}
