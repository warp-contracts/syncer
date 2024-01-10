package check

import (
	"strings"

	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/turbo"
)

// Periodically gets the current network height from warp's GW and confirms bundle is FINALIZED
type Checker struct {
	*task.Task
	monitor monitoring.Monitor

	// Communication with bundlr
	irysClient  *bundlr.Client
	turboClient *turbo.Client

	// Interactions that can be checked
	input chan *Payload

	// Finalized interactions
	Output chan *Payload
}

// Receives bundles that can be checked in bundlr
func NewChecker(config *config.Config) (self *Checker) {
	self = new(Checker)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "checker").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.Checker.WorkerPoolSize, config.Checker.WorkerQueueSize)

	return
}

func (self *Checker) WithIrysClient(client *bundlr.Client) *Checker {
	self.irysClient = client
	return self
}

func (self *Checker) WithTurboClient(client *turbo.Client) *Checker {
	self.turboClient = client
	return self
}

func (self *Checker) WithInputChannel(input chan *Payload) *Checker {
	self.input = input
	return self
}

func (self *Checker) WithMonitor(monitor monitoring.Monitor) *Checker {
	self.monitor = monitor
	return self
}

func (self *Checker) checkIrys(payload *Payload) (isFinalized bool, err error) {
	status, err := self.irysClient.GetStatus(self.Ctx, payload.BundlerTxId)
	if err != nil {
		// Update monitoring
		self.monitor.GetReport().Checker.Errors.BundrlGetStatusError.Inc()
		self.Log.WithField("tx_id", payload.BundlerTxId).WithError(err).Error("Failed to get bundle status from Irys")
		return
	}
	isFinalized = strings.EqualFold(status.Status, "FINALIZED")
	return
}

func (self *Checker) checkTurbo(payload *Payload) (isFinalized bool, err error) {
	status, err := self.turboClient.GetStatus(self.Ctx, payload.BundlerTxId)
	if err != nil {
		// Update monitoring
		self.monitor.GetReport().Checker.Errors.TurboGetStatusError.Inc()
		self.Log.WithField("tx_id", payload.BundlerTxId).WithError(err).Error("Failed to get bundle status from Turbo")
		return
	}
	isFinalized = strings.EqualFold(status.Status, "FINALIZED")
	return
}

func (self *Checker) run() error {
	// Blocks waiting for the next network height
	// Quits when the channel is closed
	for payload := range self.input {
		self.Log.WithField("id", payload.BundlerTxId).Debug("Got bundle to check")

		// Update monitoring
		self.monitor.GetReport().Checker.State.AllCheckedBundles.Inc()

		payload := payload

		self.SubmitToWorker(func() {
			var (
				err         error
				isFinalized bool
			)

			self.Log.WithField("service", payload.Service.String()).WithField("id", payload.BundlerTxId).Debug("Checking status")

			// Check if the bundle is finalized
			switch payload.Service {
			case model.BundlingServiceIrys:
				isFinalized, err = self.checkIrys(payload)
				if err != nil {
					return
				}
				if !isFinalized {
					// Update monitoring
					self.monitor.GetReport().Checker.State.IrysUnfinishedBundles.Inc()
				}
			case model.BundlingServiceTurbo:
				isFinalized, err = self.checkTurbo(payload)
				if err != nil {
					return
				}
				if !isFinalized {
					// Update monitoring
					self.monitor.GetReport().Checker.State.TurboUnfinishedBundles.Inc()
				}
			}

			if !isFinalized {
				// Update monitoring
				self.monitor.GetReport().Checker.State.UnfinishedBundles.Inc()
				return
			}

			// Update monitoring
			self.monitor.GetReport().Checker.State.FinishedBundles.Inc()

			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- payload:
			}

		})
	}

	return nil
}
