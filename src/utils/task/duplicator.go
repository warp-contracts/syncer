package task

import (
	"sync"
	"syncer/src/utils/config"
)

// Takes item from input channel and in parallel puts to all output channels.
// Payload needs to be broadcasted into all channels, otherwise it blocks
type Duplicator[In any] struct {
	*Task

	input  chan In
	output []chan In
}

func NewDuplicator[In any](config *config.Config, name string) (self *Duplicator[In]) {
	self = new(Duplicator[In])

	self.Task = NewTask(config, name).
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			for i := range self.output {
				close(self.output[i])
			}
		})

	return
}

func (self *Duplicator[In]) WithInputChannel(input chan In) *Duplicator[In] {
	self.input = input
	return self
}

func (self *Duplicator[In]) WithOutputChannels(numChannels, capacity int) *Duplicator[In] {
	for i := 0; i < numChannels; i++ {
		self.output = append(self.output, make(chan In, capacity))
	}
	self.Task = self.Task.WithWorkerPool(numChannels, 0)
	return self
}

func (self *Duplicator[In]) GetChannel(idx int) chan In {
	return self.output[idx]
}

func (self *Duplicator[In]) run() error {
	wg := sync.WaitGroup{}
	for in := range self.input {
		wg.Add(len(self.output))
		for channelIdx := range self.output {
			channelIdx := channelIdx
			self.SubmitToWorker(func() {
				select {
				case <-self.Ctx.Done():
				case self.output[channelIdx] <- in:
				}

				wg.Done()
			})
		}

		// Wait for all channels to receive data
		wg.Wait()
	}
	return nil
}