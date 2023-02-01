package monitor

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

	report map[Kind]int

	previousReport map[Kind]int
}

type Report struct {
	DbErrors                  int `json:"db"`
	TxValidationErrors        int `json:"tx_validation"`
	TxDownloadErrors          int `json:"tx_download"`
	BlockValidationErrors     int `json:"block_validation"`
	BlockDownloadErrors       int `json:"block_download"`
	PeerDownloadErrors        int `json:"peer_download"`
	NetworkInfoDownloadErrors int `json:"network_info_download"`
}

func NewMonitor() (self *Monitor) {
	self = new(Monitor)
	self.log = logger.NewSublogger("monitor")
	self.report = make(map[Kind]int)
	self.previousReport = make(map[Kind]int)
	return
}

func (self *Monitor) Increment(kind Kind) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	self.report[kind] = self.report[kind] + 1
}

func (self *Monitor) OnGet(c *gin.Context) {
	self.mtx.Lock()
	defer self.mtx.Unlock()

	status := http.StatusOK
	for k, v := range self.report {
		if self.previousReport[k] != v {
			self.log.WithField("report", self.report).Error("Error returned from monitor")
			status = http.StatusInternalServerError
			break
		}
	}

	c.JSON(status, &self.report)

	// Copy the report
	for k, v := range self.report {
		self.previousReport[k] = v
	}
}
