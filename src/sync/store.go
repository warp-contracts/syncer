package sync

import (
	"errors"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitor"
	"syncer/src/utils/task"

	"context"
	"time"

	"github.com/cenkalti/backoff/v4"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Store handles saving data to the database in na robust way.
// - groups incoming Interactions into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Store struct {
	*task.Task

	input chan *listener.Payload

	DB *gorm.DB

	monitor *monitor.Monitor
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Task = task.NewTask(config, "store").
		WithSubtaskFunc(self.run)

	return
}

func (self *Store) WithMonitor(v *monitor.Monitor) *Store {
	self.monitor = v
	return self
}

func (self *Store) WithInputChannel(v chan *listener.Payload) *Store {
	self.input = v
	return self
}

func (self *Store) WithDB(v *gorm.DB) *Store {
	self.DB = v
	return self
}

func (self *Store) insert(pendingInteractions []*model.Interaction, lastTransactionBlockHeight int64) (err error) {
	operation := func() error {
		self.Log.WithField("length", len(pendingInteractions)).Debug("Insert batch of interactions")
		err = self.DB.WithContext(self.Ctx).
			Transaction(func(tx *gorm.DB) error {
				err = self.setLastTransactionBlockHeight(self.Ctx, tx, lastTransactionBlockHeight)
				if err != nil {
					self.Log.WithError(err).Error("Failed to update last transaction block height")
					self.monitor.Report.Errors.DbLastTransactionBlockHeightError.Inc()
					return err
				}

				if len(pendingInteractions) == 0 {
					return nil
				}

				err = tx.WithContext(self.Ctx).
					Table("interactions").
					Clauses(clause.OnConflict{DoNothing: true}).
					CreateInBatches(pendingInteractions, len(pendingInteractions)).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to insert Interactions")
					self.Log.WithField("interactions", pendingInteractions).Debug("Failed interactions")
					self.monitor.Report.Errors.DbInteractionInsert.Inc()
					return err
				}
				return nil
			})
		if err == nil {
			self.monitor.Report.SyncerFinishedHeight.Store(lastTransactionBlockHeight)
			self.monitor.Report.InteractionsSaved.Add(uint64(len(pendingInteractions)))
		}
		return err
	}

	if self.IsStopping.Load() {
		return
	}

	// Expotentially increase the interval between retries
	// Never stop retrying
	// Wait at most 30s between retries
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 0
	b.MaxInterval = self.Config.StoreMaxBackoffInterval

	return backoff.Retry(operation, b)

}

// Receives data from the input channel and saves in the database
func (self *Store) run() (err error) {
	// Used to ensure data isn't stuck in Syncer for too long
	timer := time.NewTimer(self.Config.StoreMaxTimeInQueue)

	var pendingInteractions []*model.Interaction
	var lastTransactionBlockHeight int64

	insert := func() {
		err = self.insert(pendingInteractions, lastTransactionBlockHeight)
		if err != nil {
			// This is a terminal error, it already tried to retry
			self.Log.WithError(err).Error("Failed to insert data")

			// Trigger stopping
			self.Stop()
			return
		}

		// Cleanup buffer
		pendingInteractions = nil
	}

	for {
		select {
		// case <-self.CtxRunning.Done():
		// 	// Stop was requested, close the input channel
		// 	// Won't accept new data, but will process pending

		case payload, ok := <-self.input:
			if !ok {
				// The only way input channel is closed is that the Store is stopping
				// There will be no more data, insert everything there is and quit.
				insert()

				return
			}

			self.Log.WithField("height", payload.BlockHeight).WithField("num_interactions", len(pendingInteractions)).Info("Got block")
			lastTransactionBlockHeight = payload.BlockHeight

			pendingInteractions = append(pendingInteractions, payload.Interactions...)

			if len(pendingInteractions) >= self.Config.StoreBatchSize {
				insert()
			}

		case <-timer.C:
			if len(pendingInteractions) > 0 {
				self.Log.Debug("Batch timed out, trigger insert")
				insert()
			}
			timer = time.NewTimer(self.Config.StoreMaxTimeInQueue)
		}
	}
}

func (self *Store) setLastTransactionBlockHeight(ctx context.Context, tx *gorm.DB, value int64) (err error) {
	if value <= 0 {
		err = errors.New("block height too small")
		return
	}
	state := model.State{Id: 1}
	return self.DB.WithContext(ctx).Model(&state).Update("last_transaction_block_height", value).Error
}
