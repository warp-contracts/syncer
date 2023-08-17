package fixsortkey

import (
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
	"gorm.io/gorm"
)

// Gets all interactions per block from the database
// Updates sort key for each interaction
// Updates last sort key for each interaction that was pointing to the old sort key
type Processor struct {
	*task.Task
	db *gorm.DB

	input chan uint64
}

func NewProcessor(config *config.Config) (self *Processor) {
	self = new(Processor)

	self.Task = task.NewTask(config, "processor").
		WithSubtaskFunc(self.run)

	return
}

func (self *Processor) WithDB(db *gorm.DB) *Processor {
	self.db = db
	return self
}

func (self *Processor) WithInputChannel(input chan uint64) *Processor {
	self.input = input
	return self
}

func (self *Processor) run() (err error) {
	for height := range self.input {
		self.Log.WithField("height", height).Debug("Start")
	retry:
		err = self.db.WithContext(self.Ctx).
			Transaction(func(tx *gorm.DB) (err error) {
				var interactions []*model.Interaction
				// Get a batch of L1 interactions
				err = tx.Table(model.TableInteraction).
					Where("block_height = ?", height).
					Where("source=?", "arweave").
					Order("sort_key ASC").
					Find(&interactions).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get interactions")
					return
				}

				if len(interactions) == 0 {
					// No interactions in this block
					return
				}

				// Get the block hash
				for i, interaction := range interactions {
					oldSortKey := interaction.SortKey

					interactions[i].SortKey = warp.CreateSortKey(interaction.InteractionId.Bytes(), interaction.BlockHeight, interaction.BlockId.Bytes())
					self.Log.WithField("sortKey", interactions[i].SortKey).WithField("old", oldSortKey).Info("Sort key")

					if oldSortKey == interactions[i].SortKey {
						self.Log.WithField("id", interaction.ID).Info("No change needed")
						continue
					}

					err := tx.Exec(`UPDATE interactions SET sort_key=? WHERE id=?`, interactions[i].SortKey, interaction.ID).Error
					if err != nil {
						self.Log.WithError(err).Error("Failed to update sort key")
						return err
					}

					err = tx.Exec(`UPDATE interactions SET last_sort_key=? WHERE last_sort_key=?`, interactions[i].SortKey, oldSortKey).Error
					if err != nil {
						self.Log.WithError(err).Error("Failed to update last sort key")
						return err
					}
				}

				return nil
			})
		if err != nil {
			self.Log.WithError(err).WithField("height", height).Error("Failed to fetch interactions from DB")

			time.Sleep(2 * time.Second)
			goto retry
		}

		self.Log.WithField("height", height).Debug("Finish")

	}
	return
}
