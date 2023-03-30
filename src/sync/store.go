package sync

import (
	"errors"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"

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

	monitor monitoring.Monitor
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Task = task.NewTask(config, "store").
		WithSubtaskFunc(self.run)

	return
}

func (self *Store) WithMonitor(v monitoring.Monitor) *Store {
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

func (self *Store) insert(pendingInteractions []*model.Interaction, lastTransactionBlockHeight int64, lastProcessedBlockHash arweave.Base64String) (err error) {
	operation := func() error {
		self.Log.WithField("length", len(pendingInteractions)).WithField("hash", lastProcessedBlockHash.Base64()).Info("Insert batch of interactions and state")
		err = self.DB.WithContext(self.Ctx).
			Transaction(func(tx *gorm.DB) error {
				if lastTransactionBlockHeight <= 0 {
					return errors.New("block height too small")
				}
				state := model.State{
					Id:                         1,
					LastTransactionBlockHeight: lastTransactionBlockHeight,
					LastProcessedBlockHash:     lastProcessedBlockHash,
				}
				err = tx.WithContext(self.Ctx).
					Save(&state).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to update last transaction block height")
					self.monitor.GetReport().Syncer.Errors.DbLastTransactionBlockHeightError.Inc()
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
					self.monitor.GetReport().Syncer.Errors.DbInteractionInsert.Inc()
					return err
				}
				return nil
			})
		if err == nil {
			self.monitor.GetReport().Syncer.State.SyncerFinishedHeight.Store(lastTransactionBlockHeight)
			self.monitor.GetReport().Syncer.State.InteractionsSaved.Add(uint64(len(pendingInteractions)))
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
	var lastProcessedBlockHash []byte

	insert := func() {
		err = self.insert(pendingInteractions, lastTransactionBlockHeight, lastProcessedBlockHash)
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
		case payload, ok := <-self.input:
			if !ok {
				// The only way input channel is closed is that the Store is stopping
				// There will be no more data, insert everything there is and quit.
				insert()

				return
			}

			self.Log.WithField("height", payload.BlockHeight).WithField("num_interactions", len(pendingInteractions)).Info("Got block")
			lastTransactionBlockHeight = payload.BlockHeight
			lastProcessedBlockHash = payload.BlockHash

			pendingInteractions = append(pendingInteractions, payload.Interactions...)

			if len(pendingInteractions) >= self.Config.StoreBatchSize {
				insert()
			}

		case <-timer.C:
			insert()
			timer = time.NewTimer(self.Config.StoreMaxTimeInQueue)
		}
	}
}
