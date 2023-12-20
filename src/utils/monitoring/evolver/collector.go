package monitor_evolver

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct {
	monitor *Monitor

	// Run
	UpForSeconds *prometheus.Desc

	StoreDbError             *prometheus.Desc
	PollerFetchError         *prometheus.Desc
	DownloaderDownlaodError  *prometheus.Desc
	PollerSourcesFromSelects *prometheus.Desc
	StoreSourcesSaved        *prometheus.Desc
	DownloaderSourcesLoaded  *prometheus.Desc
}

func NewCollector() *Collector {
	return &Collector{
		UpForSeconds:             prometheus.NewDesc("up_for_seconds", "", nil, nil),
		StoreDbError:             prometheus.NewDesc("store_sources_saved_error", "", nil, nil),
		PollerFetchError:         prometheus.NewDesc("poller_fetch_error", "", nil, nil),
		DownloaderDownlaodError:  prometheus.NewDesc("downloader_download_error", "", nil, nil),
		PollerSourcesFromSelects: prometheus.NewDesc("poller_sources_from_selects", "", nil, nil),
		StoreSourcesSaved:        prometheus.NewDesc("store_sources_saved", "", nil, nil),
		DownloaderSourcesLoaded:  prometheus.NewDesc("downloader_sources_loaded", "", nil, nil),
	}
}

func (self *Collector) WithMonitor(m *Monitor) *Collector {
	self.monitor = m
	return self
}

func (self *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Run
	ch <- self.UpForSeconds

	ch <- self.PollerSourcesFromSelects
	ch <- self.StoreSourcesSaved
	ch <- self.DownloaderSourcesLoaded

	// Errors
	ch <- self.DownloaderDownlaodError
	ch <- self.StoreDbError
	ch <- self.PollerFetchError
}

// Collect implements required collect function for all promehteus collectors
func (self *Collector) Collect(ch chan<- prometheus.Metric) {
	// Run
	ch <- prometheus.MustNewConstMetric(self.UpForSeconds, prometheus.GaugeValue, float64(self.monitor.Report.Run.State.UpForSeconds.Load()))

	// Bundler
	ch <- prometheus.MustNewConstMetric(self.PollerSourcesFromSelects, prometheus.CounterValue, float64(self.monitor.Report.Evolver.State.PollerSourcesFromSelects.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreSourcesSaved, prometheus.CounterValue, float64(self.monitor.Report.Evolver.State.StoreSourcesSaved.Load()))
	ch <- prometheus.MustNewConstMetric(self.DownloaderSourcesLoaded, prometheus.CounterValue, float64(self.monitor.Report.Evolver.State.DownloaderSourcesLoaded.Load()))
	ch <- prometheus.MustNewConstMetric(self.DownloaderDownlaodError, prometheus.CounterValue, float64(self.monitor.Report.Evolver.Errors.DownloaderDownlaodError.Load()))
	ch <- prometheus.MustNewConstMetric(self.StoreDbError, prometheus.CounterValue, float64(self.monitor.Report.Evolver.Errors.StoreDbError.Load()))
	ch <- prometheus.MustNewConstMetric(self.PollerFetchError, prometheus.CounterValue, float64(self.monitor.Report.Evolver.Errors.PollerFetchError.Load()))

}
