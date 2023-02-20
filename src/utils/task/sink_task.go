package task

import (
	"sync"
	"syncer/src/utils/config"
	"time"

	"github.com/gammazero/deque"
)

// Task that receives data through a channel and periodically pushes to the database
type SinkTask[T comparable] struct {
	*Task

	// For synchronizing access to the queue
	mtx sync.RWMutex

	// Batch size
	batchSize int

	// Periodically called callback that processes a batch of data
	onFlush func([]T) error

	// Data about the interactions that need to be bundled
	input chan T

	// Ids of the bundle items that will be confirmed
	queue deque.Deque[T]
}

func NewSinkTask[T comparable](config *config.Config, name string) (self *SinkTask[T]) {
	self = new(SinkTask[T])
	self.Task = NewTask(config, name).
		WithSubtaskFunc(self.receive)

	return
}

func (self *SinkTask[T]) WithOnFlush(f func([]T) error) *SinkTask[T] {
	self.onFlush = f
	return self
}

func (self *SinkTask[T]) WithBatchSize(batchSize int) *SinkTask[T] {
	self.queue.SetMinCapacity(2 * uint(batchSize))
	self.batchSize = batchSize
	return self
}

func (self *SinkTask[T]) WithInputChannel(input chan T) *SinkTask[T] {
	self.input = input
	return self
}

func (self *SinkTask[T]) WithInterval(interval time.Duration) *SinkTask[T] {
	self.Task = self.Task.
		WithPeriodicSubtaskFunc(interval, self.flush)
	return self
}

func (self *SinkTask[T]) numPendingConfirmation() int {
	self.mtx.RLock()
	defer self.mtx.RUnlock()
	return self.queue.Len()
}

// Puts data into the queue
func (self *SinkTask[T]) receive() error {
	for data := range self.input {
		self.mtx.Lock()
		self.queue.PushBack(data)
		self.mtx.Unlock()
	}
	return nil
}

func (self *SinkTask[T]) flush() error {
	if self.numPendingConfirmation() > 10000 {
		self.Log.WithField("len", self.queue.Len()).Warn("Too many data in queue")
	}

	// Repeat while there's still data in the queue
	for {
		self.mtx.Lock()
		size := self.queue.Len()
		if size == 0 {
			// No data left, break the infinite loop
			self.mtx.Unlock()
			break
		}

		if size > self.batchSize {
			size = self.batchSize
		}

		// Copy data to avoid locking for too long
		batch := make([]T, 0, size)
		for i := 0; i < size; i++ {
			batch = append(batch, self.queue.PopFront())
		}
		self.mtx.Unlock()

		err := self.onFlush(batch)
		if err != nil {
			return err
		}
	}
	return nil
}
