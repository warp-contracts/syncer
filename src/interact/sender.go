package interact

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/sequencer"
	"github.com/warp-contracts/syncer/src/utils/sequencer/types"
	"github.com/warp-contracts/syncer/src/utils/task"
)

// Sends data item to sequencer
type Sender struct {
	*task.Task
	monitor monitoring.Monitor

	// Communication with bundlr
	sequencerClient *sequencer.Client

	// Interactions that can be checked
	input <-chan *Payload

	// Finalized interactions
	Output chan *Payload
}

// Receives bundles that can be checked in bundlr
func NewSender(config *config.Config) (self *Sender) {
	self = new(Sender)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "sender").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.Interactor.SenderWorkerPoolSize, config.Interactor.SenderWorkerQueueSize)

	return
}

func (self *Sender) WithClient(client *sequencer.Client) *Sender {
	self.sequencerClient = client
	return self
}

func (self *Sender) WithInputChannel(input <-chan *Payload) *Sender {
	self.input = input
	return self
}

func (self *Sender) WithMonitor(monitor monitoring.Monitor) *Sender {
	self.monitor = monitor
	return self
}

func (self *Sender) run() error {
	// Blocks waiting for the next network height
	// Quits when the channel is closed
	for payload := range self.input {

		self.Log.WithField("id", payload.DataItem.Id.Base64()).Debug("Sending data item to sequencer")

		payload := payload

		self.SubmitToWorker(func() {
			_, _, err := self.sequencerClient.Upload(self.Ctx, payload.DataItem, types.BroadcastModeSync)
			if err != nil {
				// Update monitoring
				self.monitor.GetReport().Interactor.Errors.SenderError.Inc()
				self.Log.WithField("id", payload.DataItem.Id.Base64()).WithError(err).Error("Failed to send data item to sequencer")
				return
			}

			// Update monitoring
			self.monitor.GetReport().Interactor.State.SenderSentDataItems.Inc()

			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- payload:
			}

		})
	}

	return nil
}
