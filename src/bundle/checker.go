package bundle

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/bundlr"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets the current network height from warp's GW and confirms bundle is FINALIZED
type Checker struct {
	*task.Task
	db *gorm.DB

	bundlrClient *bundlr.Client

	networkMonitor *listener.NetworkMonitor
}

func NewChecker(config *config.Config) (self *Checker) {
	self = new(Checker)

	// Arweave client
	client := arweave.NewClient(self.CtxRunning, config)

	// Gets network height from WARP's GW
	self.networkMonitor = listener.NewNetworkMonitor(config, config.Bundler.CheckerInterval).
		WithClient(client).
		WithRequiredConfirmationBlocks(0)

	self.Task = task.NewTask(config, "checker").
		WithSubtask(self.networkMonitor.Task).
		WithSubtaskFunc(self.run)

	return
}

func (self *Checker) WithClient(client *bundlr.Client) *Checker {
	self.bundlrClient = client
	return self
}

func (self *Checker) WithDB(db *gorm.DB) *Checker {
	self.db = db
	return self
}

func (self *Checker) run() error {
	// Blocks waiting for the next network height
	// Quits when the channel is closed
	for networkInfo := range self.networkMonitor.NetworkInfo {
		// Bundlr.network says it may takie 50 blocks for the tx to be finalized,
		// no need to check it sooner
		minHeightToCheck := networkInfo.Height - self.Config.Bundler.CheckerMinConfirmationBlocks

		// Each iteration handles a batch of bundles
		for {
			// This loop may take a while, so we need to check if we're still running
			if self.IsStopping.Load() {
				return nil
			}

			// Get bundles that are not finalized yet
			var bundles []model.BundleItem
			err := self.db.Model(&model.BundleItem{}).
				Select("interaction_id", "bundle_tx_id").
				Where("block_height < ?", minHeightToCheck).
				Where("status = ?", model.BundleStateUploaded).
				Order("block_height ASC").
				Limit(self.Config.Bundler.CheckerMaxBundlesPerRun).
				Preload("Interaction").
				Scan(&bundles).Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to get bundles to check")
				return nil
			}

			if len(bundles) == 0 {
				// Everything is processed,
				// break the infite loop
				break
			}

			for _, bundle := range bundles {
				// Check if the bundle is finalized
				_, err := self.bundlrClient.GetStatus(self.CtxRunning, bundle.Interaction.BundlerTxId)
				if err != nil {
					self.Log.WithError(err).Error("Failed to get bundle status")
					continue
				}

				// TODO: Update status in the database if FINALIZED
			}
		}
	}

	return nil
}
