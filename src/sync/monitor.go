package sync

import (
	"net/http"
	"sync"
	"syncer/src/utils/logger"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Rest API server
type Monitor struct {
	log *logrus.Entry
	mtx sync.Mutex

	report Report
}

type Report struct {
	DbErrors int `json:"db"`
}

func NewMonitor() (self *Monitor) {
	self = new(Monitor)
	self.log = logger.NewSublogger("monitor")
	return
}

func (self *Monitor) ReportDBError() {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.report.DbErrors += 1
}

func (self *Monitor) OnGet(c *gin.Context) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	status := http.StatusOK
	if self.report.DbErrors != 0 {
		status = http.StatusInternalServerError
	}
	c.JSON(status, &self.report)

	// Reset counters
	self.report.DbErrors = 0
}
