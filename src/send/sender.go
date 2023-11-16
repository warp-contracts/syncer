package send

import (
	"errors"

	"github.com/go-resty/resty/v2"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/bundlr/responses"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

type Sender struct {
	*task.Task
	db      *gorm.DB
	input   chan *model.DataItem
	monitor monitoring.Monitor

	// Bundling and signing
	irysClient *bundlr.Client
	signer     *bundlr.ArweaveSigner

	// Updated data items
	Output chan *model.DataItem
}

// Receives bundle items from the input channel and sends them to bundlr
func NewSender(config *config.Config, db *gorm.DB) (self *Sender) {
	var err error

	self = new(Sender)
	self.db = db

	self.Output = make(chan *model.DataItem)

	self.Task = task.NewTask(config, "sender").
		// Pool of workers that perform requests
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.Sender.BundlerNumBundlingWorkers, config.Sender.WorkerPoolQueueSize).
		WithSubtaskFunc(self.run)

	self.signer, err = bundlr.NewArweaveSigner(config.Bundlr.Wallet)
	if err != nil {
		self.Log.WithError(err).Panic("Failed to create bundlr signer")
	}

	return
}

func (self *Sender) WithClient(client *bundlr.Client) *Sender {
	self.irysClient = client
	return self
}

func (self *Sender) WithInputChannel(in chan *model.DataItem) *Sender {
	self.input = in
	return self
}

func (self *Sender) WithMonitor(monitor monitoring.Monitor) *Sender {
	self.monitor = monitor
	return self
}

func (self *Sender) run() (err error) {
	// Waits for new data items
	// Finishes when when the source of items is closed
	// It should be safe to assume all pending items are processed
	for item := range self.input {
		// Update stats
		self.monitor.GetReport().Sender.State.AllBundlesFromDb.Inc()

		// Copy the pointer so it's not overwritten in the next iteration
		item := item

		self.SubmitToWorker(func() {
			if self.IsStopping.Load() {
				// Don't start sending new items if we're stopping
				return
			}

			var (
				uploadResponse *responses.Upload
				resp           *resty.Response
				err            error
			)

			dataItem, err := self.parse(item)
			if err != nil {
				self.Log.WithError(err).
					WithField("data_item_id", item.DataItemID).
					Error("Failed parse and validate data item")
				item.State = model.BundleStateMalformed
				goto end
			}

			// Send the bundle item to the bundling service
			uploadResponse, resp, err = self.irysClient.Upload(self.Ctx, dataItem)
			if err != nil {
				if resp != nil {
					self.Log.WithError(err).
						WithField("data_item_id", item.DataItemID).
						WithField("resp", string(resp.Body())).
						WithField("url", resp.Request.URL).
						Error("Failed to upload data item to Irys")
				} else {
					self.Log.WithError(err).
						WithField("data_item_id", item.DataItemID).
						Error("Failed to upload data item to Irys, no response")
				}

				// Update stats
				self.monitor.GetReport().Sender.Errors.IrysError.Inc()

				// Bad request shouldn't be retried
				if resp != nil && resp.StatusCode() > 399 && resp.StatusCode() < 500 {
					item.State = model.BundleStateMalformed
				}

				goto end
			}

			// Check if the response is valid
			if len(uploadResponse.Id) == 0 {
				err = errors.New("Irys response has empty ID")
				self.Log.WithError(err).WithField("id", item.DataItemID).Error("Bad Irys response")
				self.monitor.GetReport().Sender.Errors.IrysError.Inc()
				return
			}

			// We'll store the JSON response
			err = item.Response.Set(uploadResponse)
			if err != nil {
				self.monitor.GetReport().Sender.Errors.IrysMarshalError.Inc()
				self.Log.WithError(err).Error("Failed to marshal response")
				return
			}

			// Update state
			item.State = model.BundleStateUploaded

			// Don't keep the data item in memory, let gc do its job
			item.DataItem.Bytes = nil

			// Update stats
			self.monitor.GetReport().Sender.State.IrysSuccess.Inc()

		end:
			// Note: Don't wait for the Ctx to be done,
			// this wouldn't save updated state to the database, which is done in the next step
			self.Output <- item
		})

	}

	return nil
}

func (self *Sender) parse(item *model.DataItem) (bundleItem *bundlr.BundleItem, err error) {
	bundleItem = new(bundlr.BundleItem)
	err = bundleItem.Unmarshal(item.DataItem.Bytes)
	if err != nil {
		self.Log.WithError(err).WithField("data_item_id", item.DataItemID).Error("Failed to unmarshal data item")
		return
	}

	err = bundleItem.Verify()
	if err != nil {
		self.Log.WithError(err).WithField("data_item_id", item.DataItemID).Error("Failed to verify data item")
		return
	}

	err = bundleItem.VerifySignature()
	if err != nil {
		self.Log.WithError(err).WithField("data_item_id", item.DataItemID).Error("Failed to verify data item signature")
		return
	}

	return
}
