package send

import (
	"github.com/jackc/pgtype"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Periodically saves the state of processed data items
type Store struct {
	*task.Hole[*model.DataItem]
	monitor        monitoring.Monitor
	db             *gorm.DB
	networkMonitor *listener.NetworkMonitor
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Hole = task.NewHole[*model.DataItem](config, "store").
		WithBatchSize(config.Sender.StoreBatchSize).
		WithOnFlush(config.Sender.StoreInterval, self.flush).
		WithBackoff(0 /* infinite retry */, config.Sender.StoreBackoffMaxInterval)

	return
}

func (self *Store) WithDB(db *gorm.DB) *Store {
	self.db = db
	return self
}

func (self *Store) WithInputChannel(input chan *model.DataItem) *Store {
	self.Hole.WithInputChannel(input)
	return self
}

func (self *Store) WithMonitor(monitor monitoring.Monitor) *Store {
	self.monitor = monitor
	return self
}

func (self *Store) WithNetworkMonitor(v *listener.NetworkMonitor) *Store {
	self.networkMonitor = v
	return self
}

func (self *Store) flush(dataItems []*model.DataItem) (err error) {
	if len(dataItems) == 0 {
		// Nothing to save
		return nil
	}

	self.Log.WithField("len", len(dataItems)).Trace("-> Saving states to DB")
	defer self.Log.WithField("len", len(dataItems)).Trace("<- Saving states to DB")

	// Network manager updates this value
	// NOTE: This can potentially block if NetworkMonitor can't get the first height
	// There's a slight delay between the actual upload an this call, but it doesn't matter
	currentBlockHeight := self.networkMonitor.GetLastNetworkInfo().Height

	// Update block height for uploaded items
	for i := range dataItems {
		if dataItems[i].State != model.BundleStateUploaded && dataItems[i].State != model.BundleStateDuplicate {
			dataItems[i].BlockHeight.Status = pgtype.Null
			continue
		}
		err = dataItems[i].BlockHeight.Set(currentBlockHeight)
		if err != nil {
			self.Log.WithError(err).Error("Failed to set block height")
			return err
		}
	}

	// Uses one transaction to do all the updates
	// NOTE: It still uses many requests to the database,
	// it should be possible to combine updates into batches, but it's not a priority for now.
	err = self.db.WithContext(self.Ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "data_item_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"state",
				"service",
				"block_height",
				"response",
			}),
		}).
		CreateInBatches(&dataItems, self.Config.Sender.StoreBatchSize).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to save bundle items, retrying...")

		// Update monitoring
		self.monitor.GetReport().Sender.Errors.ConfirmationsSavedToDbError.Inc()
		return err
	}

	// Update monitoring
	self.monitor.GetReport().Sender.State.ConfirmationsSavedToDb.Inc()

	return nil

}
