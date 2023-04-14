package monitor_contract

import (
	"math"
	"net/http"
	"syncer/src/utils/monitoring/report"
	"syncer/src/utils/task"
	"time"

	"github.com/gammazero/deque"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

// Stores and computes monitor counters
type Monitor struct {
	*task.Task

	Report report.Report

	historySize int

	collector *Collector

	// Block processing speed
	BlockHeights      *deque.Deque[int64]
	TransactionCounts *deque.Deque[uint64]
	ContractsSaved    *deque.Deque[uint64]
}

func NewMonitor() (self *Monitor) {
	self = new(Monitor)

	self.Report = report.Report{
		Run:                   &report.RunReport{},
		Contractor:            &report.ContractorReport{},
		RedisPublisher:        &report.RedisPublisherReport{},
		AppSyncPublisher:      &report.AppSyncPublisherReport{},
		NetworkInfo:           &report.NetworkInfoReport{},
		BlockDownloader:       &report.BlockDownloaderReport{},
		TransactionDownloader: &report.TransactionDownloaderReport{},
		Peer:                  &report.PeerReport{},
	}

	// Initialization
	self.Report.Run.State.StartTimestamp.Store(time.Now().Unix())

	self.collector = NewCollector().WithMonitor(self)

	self.Task = task.NewTask(nil, "monitor").
		WithPeriodicSubtaskFunc(time.Minute, self.monitorBlocks).
		WithPeriodicSubtaskFunc(time.Minute, self.monitorTransactions).
		WithPeriodicSubtaskFunc(time.Minute, self.monitorContracts)
	return
}

func (self *Monitor) GetReport() *report.Report {
	return &self.Report
}

func (self *Monitor) WithMaxHistorySize(maxHistorySize int) *Monitor {
	self.historySize = maxHistorySize

	self.BlockHeights = deque.New[int64](self.historySize)
	self.TransactionCounts = deque.New[uint64](self.historySize)
	self.ContractsSaved = deque.New[uint64](self.historySize)

	self.Report.Run.State.StartTimestamp.Store(time.Now().Unix())
	return self
}

func (self *Monitor) GetPrometheusCollector() (collector prometheus.Collector) {
	return self.collector
}

func round(f float64) float64 {
	return math.Round(f*100) / 100
}

// Measure block processing speed
func (self *Monitor) monitorBlocks() (err error) {
	loaded := self.Report.BlockDownloader.State.CurrentHeight.Load()
	if loaded == 0 {
		// Neglect the first 0
		return
	}

	self.BlockHeights.PushBack(loaded)
	if self.BlockHeights.Len() > self.historySize {
		self.BlockHeights.PopFront()
	}
	value := float64(self.BlockHeights.Back()-self.BlockHeights.Front()) / float64(self.BlockHeights.Len())

	self.Report.BlockDownloader.State.AverageBlocksProcessedPerMinute.Store(round(value))
	return
}

// Measure transaction processing speed
func (self *Monitor) monitorTransactions() (err error) {
	loaded := self.Report.TransactionDownloader.State.TransactionsDownloaded.Load()
	if loaded == 0 {
		// Neglect the first 0
		return
	}

	self.TransactionCounts.PushBack(loaded)
	if self.TransactionCounts.Len() > self.historySize {
		self.TransactionCounts.PopFront()
	}
	value := float64(self.TransactionCounts.Back()-self.TransactionCounts.Front()) / float64(self.TransactionCounts.Len())
	self.Report.TransactionDownloader.State.AverageTransactionDownloadedPerMinute.Store(round(value))
	return
}

// Measure contracts processing speed
func (self *Monitor) monitorContracts() (err error) {
	loaded := self.Report.Contractor.State.ContractsSaved.Load()
	if loaded == 0 {
		// Neglect the first 0
		return
	}

	self.ContractsSaved.PushBack(loaded)
	if self.ContractsSaved.Len() > self.historySize {
		self.ContractsSaved.PopFront()
	}
	value := float64(self.ContractsSaved.Back()-self.ContractsSaved.Front()) / float64(self.ContractsSaved.Len())
	self.Report.Contractor.State.AverageContractsSavedPerMinute.Store(round(value))
	return
}

func (self *Monitor) IsOK() bool {
	now := time.Now().Unix()
	if now-self.Report.Run.State.StartTimestamp.Load() < 300 {
		return true
	}

	// Syncer is operational long enough, check stats
	return self.Report.BlockDownloader.State.AverageBlocksProcessedPerMinute.Load() > 0.1
}

func (self *Monitor) OnGetState(c *gin.Context) {
	// pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

	// Fill data
	self.Report.BlockDownloader.State.BlocksBehind.Store(int64(self.Report.NetworkInfo.State.ArweaveCurrentHeight.Load()) - self.Report.BlockMonitor.State.SyncerCurrentHeight.Load())
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
