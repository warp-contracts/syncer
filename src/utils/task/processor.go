package task

import (
	"syncer/src/utils/config"

	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/gammazero/deque"
)

// Store handles saving data to the database in na robust way.
// - groups incoming Interactions into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Processor[In any, Out any] struct {
	*Task

	// Channel for the data to be processed
	input chan In

	// Called for each incoming data
	onProcess func(In) ([]Out, error)

	// Periodically called to handle a batch of processed data
	onFlush func(deque.Deque[Out]) error

	// Queue for the processed data
	queue deque.Deque[Out]

	// Batch size that will trigger the onFlush function
	batchSize int

	// Flush interval
	flushInterval time.Duration

	// Max times between flush retries
	maxBackoffInterval time.Duration
}

func NewProcessor[In any, Out any](config *config.Config, name string) (self *Processor[In, Out]) {
	self = new(Processor[In, Out])

	self.Task = NewTask(config, name).
		WithSubtaskFunc(self.run)

	return
}

func (self *Processor[In, Out]) WithBatchSize(batchSize int) *Processor[In, Out] {
	self.queue.SetMinCapacity(2 * uint(batchSize))
	self.batchSize = batchSize
	return self
}

func (self *Processor[In, Out]) WithInputChannel(v chan In) *Processor[In, Out] {
	self.input = v
	return self
}

func (self *Processor[In, Out]) WithOnFlush(interval time.Duration, f func(deque.Deque[Out]) error) *Processor[In, Out] {
	self.flushInterval = interval
	self.onFlush = f
	return self
}

func (self *Processor[In, Out]) WithOnProcess(f func(In) ([]Out, error)) *Processor[In, Out] {
	self.onProcess = f
	return self
}

func (self *Processor[In, Out]) WithMaxBackoffInterval(v time.Duration) *Processor[In, Out] {
	self.maxBackoffInterval = v
	return self
}

func (self *Processor[In, Out]) flush() (err error) {
	// Copy data to avoid locking for too long
	// batch := make([]Out, 0, self.queue.Len())
	// for i := 0; i < self.queue.Len(); i++ {
	// 	batch = append(batch, self.queue.PopFront())
	// }

	// Expotentially increase the interval between retries
	// Never stop retrying
	// Wait at most maxBackoffInterval between retries
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 0
	b.MaxInterval = self.maxBackoffInterval

	return backoff.Retry(func() error {
		err := self.onFlush(self.queue)
		if err != nil {
			return err
		}
		self.queue.Clear()
		return nil
	}, b)

}

// Receives data from the input channel and saves in the database
func (self *Processor[In, Out]) run() (err error) {
	// Used to ensure data isn't stuck in Processor for too long
	timer := time.NewTimer(self.flushInterval)

	for {
		select {
		case in, ok := <-self.input:
			if !ok {
				// The only way input channel is closed is that the Processor's source is stopping
				// There will be no more data, flush everything there is and quit.
				self.flush()

				return
			}

			data, err := self.onProcess(in)
			if err != nil {
				continue
			}

			// Cache the processed data
			for _, d := range data {
				self.queue.PushBack(d)
			}

			if self.queue.Len() >= self.batchSize {
				self.flush()
			}

		case <-timer.C:
			// Flush is called even if the queue is empty
			self.flush()
			timer = time.NewTimer(self.flushInterval)
		}
	}
}
