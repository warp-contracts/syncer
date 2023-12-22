package monitor_interactor

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	StartTimestamp *prometheus.Desc
	UpForSeconds   *prometheus.Desc

	// Interact
	CheckerDelay   *prometheus.Desc
	CheckerDbError *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		// Run
		StartTimestamp: prometheus.NewDesc("start_timestamp", "", nil, nil),
		UpForSeconds:   prometheus.NewDesc("up_for_seconds", "", nil, nil),

		// Interactor
		CheckerDelay:   prometheus.NewDesc("checker_delay", "", nil, nil),
		CheckerDbError: prometheus.NewDesc("checker_db_error", "", nil, nil),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Run
	ch <- self.UpForSeconds
	ch <- self.StartTimestamp

	// Interactor
	ch <- self.CheckerDelay
	ch <- self.CheckerDbError
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))
	ch <- prometheus.MustNewConstMetric(self.StartTimestamp, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.StartTimestamp.Load()))

	// Interactor
	ch <- prometheus.MustNewConstMetric(self.CheckerDelay, prometheus.GaugeValue, float64(self.monitor.Report.Interactor.State.CheckerDelay.Load()))
	ch <- prometheus.MustNewConstMetric(self.CheckerDbError, prometheus.CounterValue, float64(self.monitor.Report.Interactor.Errors.CheckerDbError.Load()))
}
