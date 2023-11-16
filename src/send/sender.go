package send

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	"errors"
	"math/rand"

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
	rand    *rand.Rand
	db      *gorm.DB
	input   chan *model.DataItem
	monitor monitoring.Monitor

	// Bundling and signing
	bundlrClient *bundlr.Client
	signer       *bundlr.ArweaveSigner

	// Ids of successfully bundled interactions
	Output chan *model.DataItem
}

// Receives bundle items from the input channel and sends them to bundlr
func NewSender(config *config.Config, db *gorm.DB) (self *Sender) {
	var err error

	self = new(Sender)
	self.db = db

	// Seed random generator
	var b [8]byte
	_, err = crypto_rand.Read(b[:])
	if err != nil {
		self.Log.WithError(err).Panic("Cannot seed math/rand package with cryptographically secure random number generator")
	}
	self.rand = rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(b[:]))))

	self.Output = make(chan *model.DataItem)

	self.Task = task.NewTask(config, "sender").
		// Pool of workers that perform requests to bundlr.
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
	self.bundlrClient = client
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

			bundleItem, err := self.parse(item)
			if err != nil {
				self.Log.WithError(err).
					WithField("data_item_id", item.DataItemID).
					Error("Failed parse and validate data item")
				item.State = model.BundleStateMalformed
				goto end
			}

			// self.Log.WithField("id", item.InteractionID).Trace("Sending interaction to Bundlr")
			// Send the bundle item to bundlr
			uploadResponse, resp, err = self.bundlrClient.Upload(self.Ctx, bundleItem)
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
						Error("Failed to upload interaction to Irys, no response")
				}

				// Update stats
				self.monitor.GetReport().Sender.Errors.BundrlError.Inc()

				// Bad request shouldn't be retried
				if resp != nil && resp.StatusCode() > 399 && resp.StatusCode() < 500 {
					item.State = model.BundleStateMalformed
				}

				goto end
			}

			// Check if the response is valid
			if len(uploadResponse.Id) == 0 {
				err = errors.New("Bundlr response has empty ID")
				self.Log.WithError(err).WithField("id", item.DataItemID).Error("Bad bundlr response")
				self.monitor.GetReport().Sender.Errors.BundrlError.Inc()
				return
			}

			// We'll store the JSON response
			err = item.Response.Set(uploadResponse)
			if err != nil {
				self.monitor.GetReport().Sender.Errors.BundrlMarshalError.Inc()
				self.Log.WithError(err).Error("Failed to marshal response")
				return
			}

			// Update state
			item.State = model.BundleStateUploaded

			// Update stats
			self.monitor.GetReport().Sender.State.BundlrSuccess.Inc()

			if item.State == model.BundleStateUploading {
				// Sending should be retried, no need to update the state in the database
				return
			}

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
