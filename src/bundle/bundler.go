package bundle

import (
	"encoding/json"
	"math/rand"
	"syncer/src/utils/arweave"
	"syncer/src/utils/bundlr"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"
	"syncer/src/utils/tool"
	"time"

	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

type Bundler struct {
	*task.Task
	db      *gorm.DB
	input   chan *model.BundleItem
	monitor monitoring.Monitor

	// Mutex for accessing the random generator
	seedingTime time.Time
	rand        *rand.Rand

	// Bundling and signing
	bundlrClient *bundlr.Client
	signer       *bundlr.Signer

	// Ids of successfully bundled interactions
	Output chan *Confirmation
}

// Receives bundle items from the input channel and sends them to bundlr
func NewBundler(config *config.Config, db *gorm.DB) (self *Bundler) {
	self = new(Bundler)
	self.db = db
	self.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	self.seedingTime = time.Now()

	self.Output = make(chan *Confirmation)

	self.Task = task.NewTask(config, "bundler").
		// Pool of workers that perform requests to bundlr.
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.Bundler.BundlerNumBundlingWorkers, config.Bundler.WorkerPoolQueueSize).
		WithSubtaskFunc(self.run)

	var err error
	self.signer, err = bundlr.NewSigner(config.Bundlr.Wallet)
	if err != nil {
		self.Log.WithError(err).Panic("Failed to create bundlr signer")
	}

	return
}

func (self *Bundler) WithClient(client *bundlr.Client) *Bundler {
	self.bundlrClient = client
	return self
}

func (self *Bundler) WithInputChannel(in chan *model.BundleItem) *Bundler {
	self.input = in
	return self
}

func (self *Bundler) WithMonitor(monitor monitoring.Monitor) *Bundler {
	self.monitor = monitor
	return self
}

func (self *Bundler) run() (err error) {
	// Waits for new interactions to bundle
	// Finishes when when the source of items is closed
	// It should be safe to assume all pending items are processed
	for item := range self.input {
		// Re-seed random generator
		if time.Since(self.seedingTime) > 24*time.Minute {
			self.rand.Seed(time.Now().UnixNano())
			self.seedingTime = time.Now()
		}

		// Update stats
		self.monitor.GetReport().Bundler.State.AllBundlesFromDb.Inc()

		// Copy the pointer so it's not overwritten in the next iteration
		item := item
		self.SubmitToWorker(func() {
			// Fill bundle item
			bundleItem := new(bundlr.BundleItem)

			if item.Transaction.Status != pgtype.Present {
				// Data needed for creating the bundle isn't present
				// Mark it as uploaded, so it's not processed again
				return
			}

			// Anchor is needed to avoid problem with same data being uploaded multiple times in Data field
			// Bundlr rejects such transaction with error like "Transaction ... already received"
			bundleItem.Anchor = make([]byte, 32)
			n, err := self.rand.Read(bundleItem.Anchor)
			if n != 32 {
				self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Failed to generate anchor, will retry later.")
				return
			}
			if err != nil {
				self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Error when generating anchor, will retry later.")
				return
			}

			// Put transaction into the data field
			data, err := item.Transaction.MarshalJSON()
			if err != nil {
				self.Log.WithError(err).WithField("id", item.InteractionID).Error("Failed to get interaction data")
				return
			}
			bundleItem.Data = arweave.Base64String(tool.MinifyJSON(data))

			tagBytes, err := item.Tags.MarshalJSON()
			if err != nil {
				self.Log.WithError(err).WithField("len", len(tagBytes)).WithField("id", item.InteractionID).Error("Failed to get transaction tags")
				return
			}

			err = json.Unmarshal(tagBytes, &bundleItem.Tags)
			if err != nil {
				self.Log.WithError(err).WithField("len", len(tagBytes)).WithField("id", item.InteractionID).Error("Failed to unmarshal transaction tags")
				return
			}

			// self.Log.WithField("id", item.InteractionID).Trace("Sending interaction to Bundlr")
			// Send the bundle item to bundlr
			uploadResponse, resp, err := self.bundlrClient.Upload(self.Ctx, self.signer, bundleItem)
			if err != nil {
				if resp != nil {
					self.Log.WithError(err).
						WithField("id", item.InteractionID).
						WithField("resp", string(resp.Body())).
						WithField("req", resp.Request.Body).
						WithField("url", resp.Request.URL).
						Error("Failed to upload interaction to Bundlr")
				} else {
					self.Log.WithError(err).
						WithField("id", item.InteractionID).
						WithField("req", resp.Request.Body).
						WithField("url", resp.Request.URL).
						Error("Failed to upload interaction to Bundlr, no response")
				}
				// Update stats
				self.monitor.GetReport().Bundler.Errors.BundrlError.Inc()

				// Bad request shouldn't be retried
				if resp != nil && resp.StatusCode() > 399 && resp.StatusCode() < 500 {
					err := self.db.Model(&model.BundleItem{
						InteractionID: item.InteractionID,
					}).
						Where("state = ?", model.BundleStateUploading).
						Updates(model.BundleItem{
							State: model.BundleStateMalformed,
						}).
						Error
					if err != nil {
						self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Failed to update bundle item state")
					}
				}

				return
			}

			// We'll store the JSON response
			response, err := json.Marshal(uploadResponse)
			if err != nil {
				self.monitor.GetReport().Bundler.Errors.BundrlMarshalError.Inc()
				self.Log.WithError(err).Error("Failed to marshal response")
				return
			}

			// Update stats
			self.monitor.GetReport().Bundler.State.BundlrSuccess.Inc()

			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &Confirmation{
				InteractionID: item.InteractionID,
				BundlerTxID:   uploadResponse.Id,
				Response:      pgtype.JSONB{Bytes: response, Status: pgtype.Present},
			}:
			}
		})

	}

	return nil
}
