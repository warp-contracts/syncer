package bundle

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/bundlr"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"

	"github.com/jackc/pgtype"
	"gorm.io/gorm"
)

type Bundler struct {
	*task.Task
	db          *gorm.DB
	bundleItems chan *model.BundleItem

	// Bundling and signing
	bundlrClient *bundlr.Client
	signer       *bundlr.Signer

	// Ids of successfully bundled interactions
	Bundled chan *Confirmation
}

// Receives bundle items from the input channel and sends them to bundlr
func NewBundler(config *config.Config, db *gorm.DB) (self *Bundler) {
	self = new(Bundler)
	self.db = db
	self.Bundled = make(chan *Confirmation)

	self.Task = task.NewTask(config, "bundler").
		// Pool of workers that perform requests to bundlr.
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.Bundler.PollerMaxParallelQueries).
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
	self.bundleItems = in
	return self
}

func (self *Bundler) run() (err error) {
	// Waits for new set of interactions to bundle
	// Finishes when when the source of items is closed
	// It should be safe to assume all pending items are processed
	for item := range self.bundleItems {
		// Copy the pointer so it's not overwritten in the next iteration
		item := item
		self.Workers.Submit(func() {
			// Fill bundle item
			bundleItem := new(bundlr.BundleItem)

			if item.Transaction.Status != pgtype.Present {
				// Data neede for creating the bundle isn't present
				// Mark it as uploaded, so it's not processed again
				return
			}

			self.Log.WithField("id", item.InteractionID).Debug("Sending interaction to Bundlr")
			data, err := item.Transaction.MarshalJSON()
			if err != nil {
				self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Failed to get interaction data")
				return
			}

			bundleItem.Data = arweave.Base64String(data)

			// Send the bundle item to bundlr
			resp, err := self.bundlrClient.Upload(self.Ctx, self.signer, bundleItem)
			if err != nil {
				self.Log.WithError(err).WithField("id", item.InteractionID).Warn("Failed to upload interaction to Bundlr")
			}

			select {
			case <-self.Ctx.Done():
				return
			case self.Bundled <- &Confirmation{
				InteractionID: item.InteractionID,
				BundlerTxID:   resp.Id,
			}:
			}
		})

	}

	return nil
}
