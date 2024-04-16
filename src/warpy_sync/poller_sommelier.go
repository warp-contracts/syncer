package warpy_sync

import (
	"context"
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

type PollerSommelier struct {
	*task.Task

	db      *gorm.DB
	monitor monitoring.Monitor

	Output chan *[]InteractionPayload

	input chan uint64
}

func NewPollerSommelier(config *config.Config) (self *PollerSommelier) {
	self = new(PollerSommelier)

	self.Output = make(chan *[]InteractionPayload, config.WarpySyncer.PollerSommelierChannelBufferLength)

	self.Task = task.NewTask(config, "poller_sommelier").
		WithSubtaskFunc(self.handleNew).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *PollerSommelier) WithDB(db *gorm.DB) *PollerSommelier {
	self.db = db
	return self
}

func (self *PollerSommelier) WithMonitor(monitor monitoring.Monitor) *PollerSommelier {
	self.monitor = monitor
	return self
}

func (self *PollerSommelier) WithInputChannel(v chan uint64) *PollerSommelier {
	self.input = v
	return self
}

func (self *PollerSommelier) handleNew() (err error) {
	for block := range self.input {
		block := block
		self.Log.WithField("block_height", block).Debug("Checking for new assets sums...")
		ctx, cancel := context.WithTimeout(self.Ctx, self.Config.WarpySyncer.PollerSommelierTimeout)
		defer cancel()

		var AssetsSums []struct {
			FromAddress string
			Sum         float64
		}
		err = self.db.WithContext(ctx).
			Raw(`SELECT from_address, 
				SUM(assets) 
				FROM warpy_syncer_assets 
				WHERE timestamp < ? group by from_address;
		`, time.Now().Unix()-self.Config.WarpySyncer.PollerSommelierSecondsForSelect).
			Scan(&AssetsSums).Error

		if err != nil {
			if err != gorm.ErrRecordNotFound {
				self.Log.WithError(err).Error("Failed to get new assets sums")
				self.monitor.GetReport().WarpySyncer.Errors.PollerSommelierFetchError.Inc()
			}
			return
		}

		if len(AssetsSums) > 0 {
			self.Log.
				WithField("count", len(AssetsSums)).
				Debug("Polled new assets sum")
		} else {
			self.Log.Debug("No new assets sum found")
		}
		interactions := make([]InteractionPayload, len(AssetsSums))

		for i, sum := range AssetsSums {
			self.monitor.GetReport().WarpySyncer.State.PollerSommelierAssetsFromSelects.Inc()
			interactions[i] = InteractionPayload{
				FromAddress: sum.FromAddress,
				Points:      int64(sum.Sum * float64(self.Config.WarpySyncer.PollerSommelierPointsBase)),
			}
		}
		select {
		case <-self.Ctx.Done():
			return
		case self.Output <- &interactions:
		}
	}

	return
}
