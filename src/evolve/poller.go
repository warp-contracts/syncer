package evolve

import (
	"context"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets new evolved contract sources which are not yet in the db
type Poller struct {
	*task.Task

	db      *gorm.DB
	monitor monitoring.Monitor

	// Evolve to be sent out
	Output chan string
}

func NewPoller(config *config.Config) (self *Poller) {
	self = new(Poller)

	self.Output = make(chan string, config.Evolver.PollerChannelBufferLength)

	self.Task = task.NewTask(config, "poller").
		WithRepeatedSubtaskFunc(config.Evolver.PollerInterval, self.handleNew).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *Poller) WithDB(db *gorm.DB) *Poller {
	self.db = db
	return self
}

func (self *Poller) WithMonitor(monitor monitoring.Monitor) *Poller {
	self.monitor = monitor
	return self
}

func (self *Poller) handleNew() (repeat bool, err error) {
	self.Log.Debug("Checking for new evolved sources...")
	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.Evolver.PollerTimeout)
	defer cancel()

	// Gets new evolved contract sources
	var evolvedContractSources []string
	err = self.db.WithContext(ctx).
		Raw(`SELECT evolve
		FROM interactions i
		LEFT JOIN contracts_src cs
		ON cs.src_tx_id = i.evolve
		WHERE evolve IS NOT NULL AND cs.src_tx_id IS NULL
		ORDER BY i.sort_key
		LIMIT ?;`, self.Config.Evolver.PollerMaxBatchSize).
		Scan(&evolvedContractSources).Error

	if err != nil {
		if err != gorm.ErrRecordNotFound {
			self.Log.WithError(err).Error("Failed to get new evolved contract sources")
			self.monitor.GetReport().Evolver.Errors.PollerFetchError.Inc()
		}
		return
	}

	if len(evolvedContractSources) > 0 {
		self.Log.
			WithField("count", len(evolvedContractSources)).
			Debug("Polled new evolved contract sources")
	} else {
		self.Log.Debug("No new evolved contract sources found")
	}

	for _, src := range evolvedContractSources {

		// Update monitoring
		self.monitor.GetReport().Evolver.State.PollerSourcesFromSelects.Inc()

		select {
		case <-self.Ctx.Done():
			return
		case self.Output <- src:
		}
	}

	if len(evolvedContractSources) != self.Config.Evolver.PollerMaxBatchSize {
		return
	}

	repeat = true
	return
}
