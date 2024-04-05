package warpy_sync

import (
	"context"
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets new evolved contract sources which are not yet in the db
type PollerSommelier struct {
	*task.Task

	db      *gorm.DB
	monitor monitoring.Monitor

	// Evolve to be sent out
	Output chan *InteractionPayload
}

func NewPollerSommelier(config *config.Config) (self *PollerSommelier) {
	self = new(PollerSommelier)

	self.Output = make(chan *InteractionPayload, config.WarpySyncer.PollerSommelierChannelBufferLength)

	self.Task = task.NewTask(config, "poller").
		WithCronSubtaskFunc(config.WarpySyncer.PollerSommelierCron, self.handleNew).
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

func (self *PollerSommelier) handleNew() (err error) {
	self.Log.Debug("Checking for new assets sums...")
	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.WarpySyncer.PollerSommelierTimeout)
	defer cancel()

	var AssetsSums []struct {
		FromAddress string
		Sum         float64
	}
	err = self.db.WithContext(ctx).
		Raw(`SELECT from_address, 
				SUM(points) 
				FROM warpy_syncer_points 
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

	for _, sum := range AssetsSums {

		self.monitor.GetReport().WarpySyncer.State.PollerSommelierAssetsFromSelects.Inc()

		select {
		case <-self.Ctx.Done():
			return
		case self.Output <- &InteractionPayload{
			FromAddress: sum.FromAddress,
			Points:      int64(sum.Sum * float64(self.Config.WarpySyncer.PollerSommelierPointsBase)),
		}:
		}
	}

	return
}
