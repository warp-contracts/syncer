package forward

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"gorm.io/gorm"
)

// Gets L1 interactions from the DB
type Fetcher struct {
	*task.Task
	db *gorm.DB

	monitor monitoring.Monitor

	input  chan uint64
	Output chan *Payload

	// Possible values: "arweave", "sequencer"
	source string
}

func NewFetcher(config *config.Config) (self *Fetcher) {
	self = new(Fetcher)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "fetcher").
		WithSubtaskFunc(self.run)

	return
}

func (self *Fetcher) WithSource(source string) *Fetcher {
	self.source = source
	return self
}

func (self *Fetcher) WithDB(db *gorm.DB) *Fetcher {
	self.db = db
	return self
}

func (self *Fetcher) WithMonitor(monitor monitoring.Monitor) *Fetcher {
	self.monitor = monitor
	return self
}

func (self *Fetcher) WithInputChannel(input chan uint64) *Fetcher {
	self.input = input
	return self
}

func (self *Fetcher) run() (err error) {
	for height := range self.input {
		offset := 0
		isFirstBatch := true
		for {
			// Fetch interactions in batches
			var interactions []*model.Interaction
			err = self.db.Table(model.TableInteraction).
				Where("block_height = ?", height).
				Where("source=?", self.source).
				Limit(self.Config.Forwarder.FetcherBatchSize).
				Offset(offset * self.Config.Forwarder.FetcherBatchSize).

				// FIXME: -----------------> IS THIS ORDERING CORRECT? <-----------------
				// This is the order DRE gets L1 interactions
				Order("id ASC").
				Find(&interactions).
				Error
			if err != nil {
				return
			}

			isLastBatch := len(interactions) < self.Config.Forwarder.FetcherBatchSize

			for i, interaction := range interactions {
				payload := &Payload{
					First:       isFirstBatch && i == 0,
					Last:        isLastBatch && i == len(interactions)-1,
					Interaction: interaction,
				}

				select {
				// NOTE: Quit only when the whole batch is processed
				// case <-self.Ctx.Done():
				// 	return
				case self.Output <- payload:
				}
			}

			isFirstBatch = false
		}
	}
	return
}
