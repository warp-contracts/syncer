package task

import (
	"context"
	"errors"
	"math"

	"github.com/cenkalti/backoff"
	"github.com/warp-contracts/syncer/src/utils/config"

	"time"

	"github.com/gammazero/deque"
)

// Processing task. Ensures flush is over before another is started:
type Hole[In any] struct {
	*Task

	// Channel for the data to be processed
	input chan In

	// Periodically called to handle a batch of processed data
	onFlush func([]In) error

	// Queue for the processed data
	queue deque.Deque[In]

	// Batch size that will trigger the onFlush function
	batchSize int

	// Flush interval
	flushInterval time.Duration

	// Max time flush should be retried. 0 means no limit.
	maxElapsedTime time.Duration

	// Max times between flush retries
	maxInterval time.Duration
}

func NewHole[In any](config *config.Config, name string) (self *Hole[In]) {
	self = new(Hole[In])

	self.Task = NewTask(config, name).
		WithSubtaskFunc(self.run)

	return
}

func (self *Hole[In]) WithBatchSize(batchSize int) *Hole[In] {
	self.batchSize = batchSize
	exp := uint(math.Round(math.Logb(float64(batchSize)))) + 1
	self.queue.SetMinCapacity(exp)
	return self
}

func (self *Hole[In]) WithInputChannel(v chan In) *Hole[In] {
	self.input = v
	return self
}

func (self *Hole[In]) WithOnFlush(interval time.Duration, f func([]In) error) *Hole[In] {
	self.flushInterval = interval
	self.onFlush = f
	return self
}

func (self *Hole[In]) WithBackoff(maxElapsedTime, maxInterval time.Duration) *Hole[In] {
	self.maxElapsedTime = maxElapsedTime
	self.maxInterval = maxInterval
	return self
}

func (self *Hole[In]) flush() (err error) {
	size := self.queue.Len()
	data := make([]In, 0, size)
	for i := 0; i < size; i++ {
		data = append(data, self.queue.PopFront())
	}

	err = NewRetry().
		WithContext(self.Ctx).
		WithMaxElapsedTime(self.maxElapsedTime).
		WithMaxInterval(self.maxInterval).
		WithOnError(func(err error, isDurationAcceptable bool) error {
			if errors.Is(err, context.Canceled) && self.IsStopping.Load() {
				// Stopping
				return backoff.Permanent(err)
			}

			self.Log.WithError(err).Error("Failed to flush data, retrying")

			return err
		}).
		Run(func() error {
			return self.onFlush(data)
		})
	if err != nil {
		self.Log.WithError(err).Error("Failed to flush data, no more retries")
		return
	}

	return
}

// Receives data from the input channel and saves in the database
func (self *Hole[In]) run() (err error) {
	// Used to ensure data isn't stuck in Processor for too long
	timer := time.NewTimer(self.flushInterval)

	for {
		select {
		case in, ok := <-self.input:
			if !ok {
				// The only way input channel is closed is that the Processor's source is stopping
				// There will be no more data, flush everything there is and quit.
				err = self.flush()
				return
			}

			self.queue.PushBack(in)

			if self.queue.Len() >= self.batchSize {
				err = self.flush()
				if err != nil {
					return err
				}
			}

		case <-timer.C:
			// Flush is called even if the queue is empty
			err = self.flush()
			if err != nil {
				return
			}
			timer = time.NewTimer(self.flushInterval)
		}
	}
}
