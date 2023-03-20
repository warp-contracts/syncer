package monitor_checker

import (
	"net/http"
	"syncer/src/utils/monitoring/report"
	"syncer/src/utils/task"

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
		Checker:     &report.CheckerReport{},
		NetworkInfo: &report.NetworkInfoReport{},
	}

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
	c.JSON(http.StatusOK, &self.Report)
}

func (self *Monitor) OnGetHealth(c *gin.Context) {
	if self.IsOK() {
		c.Status(http.StatusOK)
	} else {
		c.Status(http.StatusServiceUnavailable)
	}
}
