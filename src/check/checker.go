package check

import (
	"syncer/src/utils/bundlr"
	"syncer/src/utils/config"
	"syncer/src/utils/task"
)

// Periodically gets the current network height from warp's GW and confirms bundle is FINALIZED
type Checker struct {
	*task.Task

	// Communication with bundlr
	bundlrClient *bundlr.Client

	// Interactions that can be checked
	input chan *Payload

	// Finalized interactions
	Output chan int
}

// Receives bundles that can be checked in bundlr
func NewChecker(config *config.Config) (self *Checker) {
	self = new(Checker)

	self.Output = make(chan int)

	self.Task = task.NewTask(config, "checker").
		WithSubtaskFunc(self.run).
		WithWorkerPool(5)

	return
}

func (self *Checker) WithClient(client *bundlr.Client) *Checker {
	self.bundlrClient = client
	return self
}

func (self *Checker) WithInputChannel(input chan *Payload) *Checker {
	self.input = input
	return self
}

func (self *Checker) run() error {
	// Blocks waiting for the next network height
	// Quits when the channel is closed
	for payload := range self.input {
		payload := payload

		self.Workers.Submit(func() {
			// Check if the bundle is finalized
			status, err := self.bundlrClient.GetStatus(self.CtxRunning, payload.BundlerTxId)
			if err != nil {
				self.Log.WithError(err).Error("Failed to get bundle status")
			}

			if status.Status != "FINALIZED" {
				return
			}

			select {
			case <-self.CtxRunning.Done():
				return
			case self.Output <- payload.InteractionId:
			}
		})
	}

	return nil
}
