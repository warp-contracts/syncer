package monitor

import (
	"math"
	"net/http"
	"syncer/src/utils/task"
	"time"

	"github.com/gammazero/deque"
	"github.com/gin-gonic/gin"
)

// Stores and computes monitor counters
type Monitor struct {
	*task.Task

	Report Report

	historySize int

	// Block processing speed
	BlockHeights      *deque.Deque[int64]
	TransactionCounts *deque.Deque[uint64]
	InteractionsSaved *deque.Deque[uint64]
}

func NewMonitor() (self *Monitor) {
	self = new(Monitor)

	self.historySize = 30

	self.Task = task.NewTask(nil, "monitor").
		WithPeriodicSubtaskFunc(time.Minute, self.monitorBlocks).
		WithPeriodicSubtaskFunc(time.Minute, self.monitorTransactions).
		WithPeriodicSubtaskFunc(time.Minute, self.monitorInteractions)
	return
}

func (self *Monitor) WithMaxHistorySize(maxHistorySize int) *Monitor {
	self.historySize = maxHistorySize

	self.BlockHeights = deque.New[int64](self.historySize)
	self.TransactionCounts = deque.New[uint64](self.historySize)
	self.InteractionsSaved = deque.New[uint64](self.historySize)

	self.Report.StartTimestamp.Store(time.Now().Unix())
	return self
}

func round(f float64) float64 {
	return math.Round(f*100) / 100
}

// Measure block processing speed
func (self *Monitor) monitorBlocks() (err error) {
	loaded := self.Report.SyncerCurrentHeight.Load()
	if loaded == 0 {
		// Neglect the first 0
		return
	}

	self.BlockHeights.PushBack(loaded)
	if self.BlockHeights.Len() > self.historySize {
		self.BlockHeights.PopFront()
	}
	value := float64(self.BlockHeights.Back()-self.BlockHeights.Front()) / float64(self.BlockHeights.Len())

	self.Report.AverageBlocksProcessedPerMinute.Store(round(value))
	return
}

// Measure transaction processing speed
func (self *Monitor) monitorTransactions() (err error) {
	loaded := self.Report.TransactionsDownloaded.Load()
	if loaded == 0 {
		// Neglect the first 0
		return
	}

	self.TransactionCounts.PushBack(loaded)
	if self.TransactionCounts.Len() > self.historySize {
		self.TransactionCounts.PopFront()
	}
	value := float64(self.TransactionCounts.Back()-self.TransactionCounts.Front()) / float64(self.TransactionCounts.Len())
	self.Report.AverageTransactionDownloadedPerMinute.Store(round(value))
	return
}

// Measure Interaction processing speed
func (self *Monitor) monitorInteractions() (err error) {
	loaded := self.Report.InteractionsSaved.Load()
	if loaded == 0 {
		// Neglect the first 0
		return
	}

	self.InteractionsSaved.PushBack(loaded)
	if self.InteractionsSaved.Len() > self.historySize {
		self.InteractionsSaved.PopFront()
	}
	value := float64(self.InteractionsSaved.Back()-self.InteractionsSaved.Front()) / float64(self.InteractionsSaved.Len())
	self.Report.AverageInteractionsSavedPerMinute.Store(round(value))
	return
}

func (self *Monitor) IsSyncerOK() bool {
	now := time.Now().Unix()
	if now-self.Report.StartTimestamp.Load() < 300 {
		return true
	}

	// Syncer is operational long enough, check stats
	return self.Report.AverageBlocksProcessedPerMinute.Load() > 0.1
}

func (self *Monitor) OnGetState(c *gin.Context) {
	// pprof.Lookup("goroutine").WriteTo(os.Stdout, 1)

	self.Report.Fill()
	c.JSON(http.StatusOK, &self.Report)
}

func (self *Monitor) OnGet(c *gin.Context) {
	status := http.StatusOK
	self.Report.Fill()
	c.JSON(status, &self.Report)
}
