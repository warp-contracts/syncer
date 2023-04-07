package task

import (
	"syncer/src/utils/config"
)

// Takes item from input channel, processes it and inserts it into the output channel
type Mapper[In any, Out any] struct {
	*Task

	process func(in In) (out Out, err error)

	input  chan In
	Output chan Out
}

func NewMapper[In any, Out any](config *config.Config, name string) (self *Mapper[In, Out]) {
	self = new(Mapper[In, Out])

	self.Output = make(chan Out)

	self.Task = NewTask(config, name).
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Mapper[In, Out]) WithInputChannel(input chan In) *Mapper[In, Out] {
	self.input = input
	return self
}

func (self *Mapper[In, Out]) WithMapFunc(f func(in In) (out Out, err error)) *Mapper[In, Out] {
	self.process = f
	return self
}

func (self *Mapper[In, Out]) WithWorkerPool(maxWorkers, maxQueueSize int) *Mapper[In, Out] {
	self.Task = self.Task.WithWorkerPool(maxWorkers, maxQueueSize)
	return self
}

func (self *Mapper[In, Out]) run() error {
	for in := range self.input {
		self.SubmitToWorker(func() {
			out, err := self.process(in)
			if err != nil {
				self.Log.WithError(err).Error("Failed to process item, skipping")
			}

			select {
			case <-self.Ctx.Done():
			case self.Output <- out:
			}
		})
	}
	return nil
}
