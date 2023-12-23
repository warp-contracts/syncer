package bundle

import (
	crypto_rand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"errors"
	"math/big"
	"math/rand"

	"github.com/go-resty/resty/v2"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	irysResponses "github.com/warp-contracts/syncer/src/utils/bundlr/responses"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/tool"
	"github.com/warp-contracts/syncer/src/utils/turbo"
	turboResponses "github.com/warp-contracts/syncer/src/utils/turbo/responses"

	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

type Bundler struct {
	*task.Task
	rand    *rand.Rand
	db      *gorm.DB
	input   chan *model.BundleItem
	monitor monitoring.Monitor

	// Bundling and signing
	irysClient  *bundlr.Client
	turboClient *turbo.Client
	signer      *bundlr.ArweaveSigner

	// Ids of successfully bundled interactions
	Output chan *Confirmation
}

// Receives bundle items from the input channel and sends them to bundlr
func NewBundler(config *config.Config, db *gorm.DB) (self *Bundler) {
	var err error

	self = new(Bundler)
	self.db = db

	// Seed random generator
	var b [8]byte
	_, err = crypto_rand.Read(b[:])
	if err != nil {
		self.Log.WithError(err).Panic("Cannot seed math/rand package with cryptographically secure random number generator")
	}
	self.rand = rand.New(rand.NewSource(int64(binary.LittleEndian.Uint64(b[:]))))

	self.Output = make(chan *Confirmation)

	self.Task = task.NewTask(config, "bundler").
		// Pool of workers that perform requests to bundlr.
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.Bundler.BundlerNumBundlingWorkers, config.Bundler.WorkerPoolQueueSize).
		WithSubtaskFunc(self.run)

	self.signer, err = bundlr.NewArweaveSigner(config.Bundlr.Wallet)
	if err != nil {
		self.Log.WithError(err).Panic("Failed to create bundlr signer")
	}

	return
}

func (self *Bundler) WithIrysClient(client *bundlr.Client) *Bundler {
	self.irysClient = client
	return self
}

func (self *Bundler) WithTurboClient(client *turbo.Client) *Bundler {
	self.turboClient = client
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

func (self *Bundler) upload(dataItem *model.BundleItem, item *bundlr.BundleItem) (response []byte, resp *resty.Response, id string, err error) {
	switch model.BundlingService(dataItem.Service.String) {
	case model.BundlingServiceTurbo:
		var (
			uploadResponse *turboResponses.Upload
		)

		uploadResponse, resp, err = self.turboClient.Upload(self.Ctx, item)
		if err != nil {
			if resp != nil {
				self.Log.WithError(err).
					WithField("data_item_id", dataItem.InteractionID).
					WithField("resp", string(resp.Body())).
					WithField("code", resp.StatusCode()).
					WithField("url", resp.Request.URL).
					Error("Failed to upload data item to Turbo")
			} else {
				self.Log.WithError(err).
					WithField("data_item_id", dataItem.InteractionID).
					Error("Failed to upload data item to Turbo, no response")
			}

			return
		}

		// Check if the response is valid
		if len(uploadResponse.Id) == 0 {
			err = errors.New("Turbo response has empty ID")
			self.Log.WithError(err).WithField("id", dataItem.InteractionID).Error("Bad Turbo response")
			self.monitor.GetReport().Bundler.Errors.TurboError.Inc()
			return
		}

		// We'll store the JSON response
		response, err = json.Marshal(uploadResponse)
		if err != nil {
			self.monitor.GetReport().Bundler.Errors.TurboMarshalError.Inc()
			self.Log.WithError(err).Error("Failed to marshal response from Turbo")
			return
		}

		id = uploadResponse.Id

	case model.BundlingServiceIrys:
		var (
			uploadResponse *irysResponses.Upload
		)

		uploadResponse, resp, err = self.irysClient.Upload(self.Ctx, item)
		if err != nil {
			if resp != nil {
				self.Log.WithError(err).
					WithField("data_item_id", dataItem.InteractionID).
					WithField("resp", string(resp.Body())).
					WithField("code", resp.StatusCode()).
					WithField("url", resp.Request.URL).
					Error("Failed to upload data item to Irys")
			} else {
				self.Log.WithError(err).
					WithField("data_item_id", dataItem.InteractionID).
					Error("Failed to upload data item to Irys, no response")
			}

			return
		}

		// Check if the response is valid
		if len(uploadResponse.Id) == 0 {
			err = errors.New("Irys response has empty ID")
			self.Log.WithError(err).WithField("id", dataItem.InteractionID).Error("Bad Irys response")
			self.monitor.GetReport().Bundler.Errors.BundrlError.Inc()
			return
		}

		// We'll store the JSON response
		response, err = json.Marshal(uploadResponse)
		if err != nil {
			self.monitor.GetReport().Bundler.Errors.BundrlMarshalError.Inc()
			self.Log.WithError(err).Error("Failed to marshal response from Irys")
			return
		}

		id = uploadResponse.Id

	default:
		err = errors.New("Unknown bundling service")
		self.Log.WithError(err).WithField("service", dataItem.Service).Error("Unknown bundling service")
	}
	return
}

func (self *Bundler) setBundleProvider(item *model.BundleItem) (err error) {
	v, err := crypto_rand.Int(crypto_rand.Reader, big.NewInt(100))
	if err != nil {
		self.Log.WithError(err).Panic("Failed to generate random number")
		return
	}

	if v.Int64() < int64(self.Config.Bundlr.IrysSendProbability) {
		return item.Service.Set(model.BundlingServiceIrys)
	}

	return item.Service.Set(model.BundlingServiceTurbo)
}

func (self *Bundler) run() (err error) {
	// Waits for new interactions to bundle
	// Finishes when when the source of items is closed
	// It should be safe to assume all pending items are processed
	for item := range self.input {
		// Update stats
		self.monitor.GetReport().Bundler.State.AllBundlesFromDb.Inc()

		// Copy the pointer so it's not overwritten in the next iteration
		item := item

		self.SubmitToWorker(func() {
			if item.Transaction.Status != pgtype.Present && item.DataItem.Status != pgtype.Present {
				// Data needed for creating the bundle isn't present
				// Mark it as uploaded, so it's not processed again
				return
			}

			bundleItem, err := self.createDataItem(item)
			if err != nil {
				return
			}

			// Pick random bundling service
			err = self.setBundleProvider(item)
			if err != nil {
				return
			}

			// Send the bundle
			uploadResponse, resp, id, err := self.upload(item, bundleItem)
			if err != nil {
				if resp != nil {
					self.Log.WithError(err).
						WithField("id", item.InteractionID).
						WithField("resp", string(resp.Body())).
						// WithField("req", resp.Request.Body).
						WithField("url", resp.Request.URL).
						Error("Failed to upload interaction to Bundlr")
				} else {
					self.Log.WithError(err).
						WithField("id", item.InteractionID).
						Error("Failed to upload interaction to Bundlr, no response")
				}

				// Update stats
				switch model.BundlingService(item.Service.String) {
				case model.BundlingServiceTurbo:
					self.monitor.GetReport().Bundler.Errors.TurboError.Inc()
				case model.BundlingServiceIrys:
					self.monitor.GetReport().Bundler.Errors.BundrlError.Inc()
				}

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
			// Check if the response is valid
			if len(id) == 0 {
				err = errors.New("Bundlr response has empty ID")
				self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Bad bundlr response")
				self.monitor.GetReport().Bundler.Errors.BundrlError.Inc()
				return
			}

			// Update stats
			switch model.BundlingService(item.Service.String) {
			case model.BundlingServiceTurbo:
				self.monitor.GetReport().Bundler.State.TurboSuccess.Inc()
			case model.BundlingServiceIrys:
				self.monitor.GetReport().Bundler.State.BundrlSuccess.Inc()
			}
			self.monitor.GetReport().Bundler.State.AllSuccess.Inc()

			// Save the response
			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &Confirmation{
				InteractionID: item.InteractionID,
				BundlerTxID:   id,
				Response:      pgtype.JSONB{Bytes: uploadResponse, Status: pgtype.Present},
				Service:       item.Service,
			}:
			}
		})

	}

	return nil
}

func (self *Bundler) createDataItem(item *model.BundleItem) (bundleItem *bundlr.BundleItem, err error) {
	if len(item.DataItem.Bytes) > 0 {
		// NEW WAY OF SENDING BUNDLES
		// GW sends the data item, bundler creates a nested bundle and sends it to bundlr
		bundleItem, err = self.createNestedBundle(item)
		if err != nil {
			return
		}
	} else {
		// OLD WAY OF SENDING BUNDLES
		// GW sent transaction + tags, bundler created the data item
		bundleItem, err = self.createBundle(item)
		if err != nil {
			return
		}
	}

	// Anchor is needed to avoid problem with same data being uploaded multiple times in Data field
	// Bundlr rejects such transaction with error like "Transaction ... already received"
	bundleItem.Anchor = make([]byte, 32)
	n, err := crypto_rand.Read(bundleItem.Anchor)
	// n, err := self.rand.Read(bundleItem.Anchor)
	if n != 32 {
		self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Failed to generate anchor, will retry later.")
		return
	}
	if err != nil {
		self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Error when generating anchor, will retry later.")
		return
	}

	err = bundleItem.Sign(self.signer)
	if err != nil {
		self.Log.WithError(err).Error("Failed to sign bundle item")
		return
	}

	return
}

func (self *Bundler) createBundle(item *model.BundleItem) (bundleItem *bundlr.BundleItem, err error) {
	bundleItem = new(bundlr.BundleItem)

	// Put transaction into the data field
	data, err := item.Transaction.MarshalJSON()
	if err != nil {
		self.Log.WithError(err).WithField("id", item.InteractionID).Error("Failed to get interaction data")
		return
	}
	bundleItem.Data = arweave.Base64String(tool.MinifyJSON(data))

	// Parse tags
	bundleItem.Tags, err = self.getTags(item)
	if err != nil {
		return
	}

	return
}

func (self *Bundler) createNestedBundle(item *model.BundleItem) (bundleItem *bundlr.BundleItem, err error) {
	// Parse bundle item
	nestedBundle := new(bundlr.BundleItem)
	err = nestedBundle.Unmarshal(item.DataItem.Bytes)
	if err != nil {
		self.Log.WithError(err).WithField("id", item.InteractionID).Error("Failed to unmarshal nested bundle item")
		return
	}

	// err = nestedBundle.Verify()
	// if err != nil {
	// 	self.Log.WithError(err).WithField("id", item.InteractionID).Error("Failed to verify nested bundle item")
	// 	return
	// }

	// err = nestedBundle.VerifySignature()
	// if err != nil {
	// 	self.Log.WithError(err).WithField("id", item.InteractionID).Error("Failed to verify nested bundle item signature")
	// 	return
	// }

	// self.Log.WithField("nested", nestedBundle.String()).Debug("Nested bundle item")

	bundleItem = new(bundlr.BundleItem)

	// Tags stored in the bundle_items table
	bundleItem.Tags, err = self.getTags(item)
	if err != nil {
		return
	}

	// Additional tags
	tags := bundlr.Tags{
		{
			Name:  "Bundle-Format",
			Value: "binary",
		},
		{
			Name:  "Bundle-Version",
			Value: "2.0.0",
		},
		{
			Name:  "App-Name",
			Value: "Warp",
		},
		{
			Name:  "Action",
			Value: "WarpInteraction",
		},
	}
	bundleItem.Tags = append(bundleItem.Tags, tags...)

	// Nest the bundle
	err = bundleItem.NestBundles([]*bundlr.BundleItem{nestedBundle})
	if err != nil {
		self.Log.WithError(err).WithField("id", item.InteractionID).Error("Failed to nest bundle")
		return
	}

	return
}

func (self *Bundler) getTags(item *model.BundleItem) (tags bundlr.Tags, err error) {
	tags = make(bundlr.Tags, 0, 10)

	tagBytes, err := item.Tags.MarshalJSON()
	if err != nil {
		self.Log.WithError(err).WithField("len", len(tagBytes)).WithField("id", item.InteractionID).Error("Failed to get transaction tags")
		return
	}

	// Accept {} as empty tags
	if len(tagBytes) == 2 && string(tagBytes) == "{}" {
		tagBytes = []byte("[]")
	}

	err = json.Unmarshal(tagBytes, &tags)
	if err != nil {
		self.Log.WithError(err).WithField("len", len(tagBytes)).WithField("id", item.InteractionID).Error("Failed to unmarshal transaction tags")
		return
	}

	return
}
