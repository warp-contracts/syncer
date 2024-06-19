package warpy_sync

import (
	"context"
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

type PollerDeposit struct {
	*task.Task

	db      *gorm.DB
	monitor monitoring.Monitor

	Output chan *[]InteractionPayload

	input chan uint64
}

func NewPollerDeposit(config *config.Config) (self *PollerDeposit) {
	self = new(PollerDeposit)

	self.Output = make(chan *[]InteractionPayload, config.WarpySyncer.PollerDepositChannelBufferLength)

	self.Task = task.NewTask(config, "poller_deposit").
		WithSubtaskFunc(self.handleNew).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *PollerDeposit) WithDB(db *gorm.DB) *PollerDeposit {
	self.db = db
	return self
}

func (self *PollerDeposit) WithMonitor(monitor monitoring.Monitor) *PollerDeposit {
	self.monitor = monitor
	return self
}

func (self *PollerDeposit) WithInputChannel(v chan uint64) *PollerDeposit {
	self.input = v
	return self
}

func (self *PollerDeposit) handleNew() (err error) {
	for block := range self.input {
		block := block
		self.Log.WithField("block_height", block).Debug("Checking for new assets sums...")
		ctx, cancel := context.WithTimeout(self.Ctx, self.Config.WarpySyncer.PollerDepositTimeout)
		defer cancel()

		var AssetsSums []struct {
			FromAddress string
			Sum         float64
		}
		err = self.db.WithContext(ctx).
			Raw(`SELECT from_address, 
				SUM(assets) 
				FROM warpy_syncer_assets 
				WHERE timestamp < ? AND chain = ? AND protocol = ?
				AND from_address = ? 
				group by from_address;
		`, time.Now().Unix()-self.Config.WarpySyncer.PollerDepositSecondsForSelect,
				self.Config.WarpySyncer.SyncerChain, self.Config.WarpySyncer.SyncerProtocol, "0x64937ab314bc1999396De341Aa66897C30008852").
			Scan(&AssetsSums).Error

		if err != nil {
			if err != gorm.ErrRecordNotFound {
				self.Log.WithError(err).Error("Failed to get new assets sums")
				self.monitor.GetReport().WarpySyncer.Errors.PollerDepositFetchError.Inc()
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
			self.monitor.GetReport().WarpySyncer.State.PollerDepositAssetsFromSelects.Inc()
			interactions[i] = InteractionPayload{
				FromAddress: sum.FromAddress,
				Points:      int64(sum.Sum * float64(self.Config.WarpySyncer.PollerDepositPointsBase)),
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
