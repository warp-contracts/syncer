package check

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets the current network height from warp's GW and confirms bundle is FINALIZED
type Poller struct {
	*task.Task
	db *gorm.DB

	input  chan *arweave.NetworkInfo
	Output chan *Payload
}

type Payload struct {
	InteractionId int
	BundlerTxId   string
}

// For every network height, fetches unfinished bundles
func NewPoller(config *config.Config) (self *Poller) {
	self = new(Poller)

	self.Task = task.NewTask(config, "poller").
		WithSubtaskFunc(self.run)

	return
}

func (self *Poller) WithDB(db *gorm.DB) *Poller {
	self.db = db
	return self
}

func (self *Poller) WithInputChannel(input chan *arweave.NetworkInfo) *Poller {
	self.input = input
	return self
}

func (self *Poller) run() error {
	// Blocks waiting for the next network height
	// Quits when the channel is closed
	for networkInfo := range self.input {
		// Bundlr.network says it may takie 50 blocks for the tx to be finalized,
		// no need to check it sooner
		minHeightToCheck := networkInfo.Height - self.Config.Checker.MinConfirmationBlocks

		// Each iteration handles a batch of bundles
		for {
			// This loop may take a while, so we need to check if we're still running
			if self.IsStopping.Load() {
				return nil
			}

			// Get bundles that are not finalized yet
			var interactions []model.Interaction
			err := self.db.Model(&model.Interaction{}).
				Select("id", "bundler_tx_id").
				Joins("JOIN bundle_items ON interactions.id = bundle_items.interaction_id").
				Where("bundle_items.block_height < ?", minHeightToCheck).
				Where("bundle_items.state = ?", model.BundleStateUploaded).
				Order("bundle_items.block_height ASC").
				Limit(self.Config.Checker.MaxBundlesPerRun).
				Preload("Interaction").
				Scan(&interactions).Error
			if err != nil {
				self.Log.WithError(err).Error("Failed to get bundles to check")
				return nil
			}

			if len(interactions) == 0 {
				// Everything is processed,
				// break the infite loop
				break
			}

			for _, interaction := range interactions {
				select {
				case <-self.CtxRunning.Done():
					return nil
				case self.Output <- &Payload{
					InteractionId: interaction.ID,
					BundlerTxId:   interaction.BundlerTxId,
				}:
				}
			}
		}
	}

	return nil
}
