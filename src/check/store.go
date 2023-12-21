package check

import (
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically saves the states
// SinkTask handles caching data and periodically calling flush function
type Store struct {
	*task.Hole[*Payload]
	db      *gorm.DB
	monitor monitoring.Monitor
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Hole = task.NewHole[*Payload](config, "store").
		WithOnFlush(5*time.Second, self.flush).
		WithBatchSize(50).
		WithBackoff(10*time.Minute, 10*time.Second)

	return
}

func (self *Store) WithDB(db *gorm.DB) *Store {
	self.db = db
	return self
}

func (self *Store) WithInputChannel(input chan *Payload) *Store {
	self.Hole = self.Hole.WithInputChannel(input)
	return self
}

func (self *Store) WithMonitor(monitor monitoring.Monitor) *Store {
	self.monitor = monitor
	return self
}

func (self *Store) flush(payloads []*Payload) error {
	if len(payloads) == 0 {
		return nil
	}

	// Create lists of ids for both tables
	bundleItemIds := make([]int, 0, len(payloads))
	dataItemIds := make([]string, 0, len(payloads))
	for _, payload := range payloads {
		switch payload.Table {
		case model.TableBundleItem:
			bundleItemIds = append(bundleItemIds, payload.InteractionId)
		case model.TableDataItem:
			dataItemIds = append(dataItemIds, payload.BundlerTxId)
		}
	}

	self.Log.WithField("len", len(payloads)).Debug("Saving checked states")
	err := self.db.Transaction(func(tx *gorm.DB) (err error) {
		if len(bundleItemIds) > 0 {
			err = self.db.Model(&model.BundleItem{}).
				Where("interaction_id IN ?", bundleItemIds).
				Update("state", model.BundleStateOnArweave).
				Error
			if err != nil {
				return
			}
		}

		if len(dataItemIds) > 0 {
			err = self.db.Model(&model.DataItem{}).
				Where("data_item_id IN ?", dataItemIds).
				Update("state", model.BundleStateOnArweave).
				Error
			if err != nil {
				return
			}
		}
		return
	})

	if err != nil {
		self.Log.WithError(err).Error("Failed to update bundle/data item state")

		// Update monitoring
		self.monitor.GetReport().Checker.Errors.DbStateUpdateError.Inc()
		return err
	}

	// Update monitoring
	self.monitor.GetReport().Checker.State.DbStateUpdated.Inc()

	return nil
}
