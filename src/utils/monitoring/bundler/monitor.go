package monitor_bundler

import (
	"net/http"
	"syncer/src/utils/monitoring/report"
	"syncer/src/utils/task"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// Stores and computes monitor counters
type Monitor struct {
	*task.Task

	Report    report.Report
	collector *Collector
}

func NewMonitor() (self *Monitor) {
	self = new(Monitor)

	self.Report = report.Report{
		Run:         &report.RunReport{},
		Bundler:     &report.BundlerReport{},
		NetworkInfo: &report.NetworkInfoReport{},
	}

	// Initialization
	self.Report.Run.State.StartTimestamp.Store(time.Now().Unix())

	self.collector = NewCollector().WithMonitor(self)

	self.Task = task.NewTask(nil, "monitor")
	return
}

func (self *Monitor) GetReport() *report.Report {
	return &self.Report
}

func (self *Monitor) GetPrometheusCollector() (collector prometheus.Collector) {
	return self.collector
}

func (self *Monitor) IsOK() bool {
	return true
}

func (self *Monitor) OnGetState(c *gin.Context) {
	self.Report.Run.State.UpForSeconds.Store(uint64(time.Now().Unix() - self.Report.Run.State.StartTimestamp.Load()))

	c.JSON(http.StatusOK, &self.Report)
}

func (self *Monitor) OnGetHealth(c *gin.Context) {
	if self.IsOK() {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusServiceUnavailable)
	}
}
