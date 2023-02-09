package bundle

import (
	"sync"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

type BundlerManager struct {
	*task.Task
	db          *gorm.DB
	bundleItems chan []model.BundleItem
}

// Main class that orchestrates main syncer functionalities
func NewBundlerManager(config *config.Config, db *gorm.DB) (self *BundlerManager) {
	self = new(BundlerManager)
	self.db = db

	self.Task = task.NewTask(config, "bundler-manager").
		// Pool of workers that perform requests to bundlr.
		// It's possible to run multiple requests in parallel.
		// We're limiting the number of parallel requests with the number of workers.
		WithWorkerPool(config.BundlerManagerNudWorkers).
		WithSubtaskFunc(self.run)

	return
}

func (self *BundlerManager) WithInputChannel(in chan []model.BundleItem) *BundlerManager {
	self.bundleItems = in
	return self
}

func (self *BundlerManager) run() (err error) {
	// Waits for new set of interactions to bundle
	// Finishes when when the source of items is closed
	// It should be safe to assume all pending items are processed
	for items := range self.bundleItems {
		var wg sync.WaitGroup
		wg.Add(len(items))
		for _, item := range items {
			item := item
			self.Workers.Submit(func() {
				self.Log.WithField("id", item.InteractionID).Debug("Sending interaction to Bundlr")
				wg.Done()
			})
		}

		// Wait for all bundles to be sent
		wg.Wait()

		// Interaction ids in one slice
		interactionIds := make([]int, len(items))
		for i, item := range items {
			interactionIds[i] = item.InteractionID
		}

		// Mark successfuly sent bundles as UPLOADED
		err = self.db.Model(&model.BundleItem{}).
			Where("interaction_id IN ?", interactionIds).
			Where("state = ?", model.BundleStateUploading).
			Update("state", model.BundleStateUploaded).
			Error
		if err != nil {
			// TODO: Is there anything else we can do?
			self.Log.WithError(err).Error("Failed to mark bundle item as uploaded to Bundlr")
		}
	}

	return nil
}
