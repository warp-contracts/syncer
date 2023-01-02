package sync

import (
	"sync/atomic"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syncer/src/utils/model"

	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Store handles saving data to the database in na robust way.
// - groups incoming Interactions into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Store struct {
	Ctx    context.Context
	cancel context.CancelFunc

	config *config.Config
	log    *logrus.Entry
	input  chan *Payload
	DB     *gorm.DB

	stopChannel chan bool
	isStopping  *atomic.Bool
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)
	self.log = logger.NewSublogger("store")
	self.config = config

	// Store's context, active as long as there's anything running in Store
	self.Ctx, self.cancel = context.WithCancel(context.Background())
	self.Ctx = common.SetConfig(self.Ctx, config)

	// Internal channel for closing the underlying goroutine
	self.stopChannel = make(chan bool, 1)

	// Incoming interactions channel
	self.input = make(chan *Payload)

	// Variable used for avoiding stopping Store two times upon panics/errors and saving to stopped Store
	self.isStopping = &atomic.Bool{}
	return
}

func (self *Store) connect() (db *gorm.DB, err error) {
	if self.DB != nil {
		// Just check the connection
		err = model.Ping(self.Ctx, self.DB)
		if err == nil {
			return self.DB, nil
		}

		self.log.WithError(err).Warn("Ping failed, restarting connection")
	}
	return model.NewConnection(self.Ctx)
}

func (self *Store) Start() (err error) {
	self.log.Info("Starting Store...")

	// Connection to the database
	self.DB, err = self.connect()
	if err != nil {
		return
	}

	go func() {
		defer func() {
			// run() finished, so it's time to cancel Store's context
			// NOTE: This should be the only place self.Ctx is cancelled
			self.cancel()

			var err error
			if p := recover(); p != nil {
				switch p := p.(type) {
				case error:
					err = p
				default:
					err = fmt.Errorf("%s", p)
				}
				self.log.WithError(err).Error("Panic in Store. Stopping.")
				panic(p)
			}
		}()
		err := self.run()
		if err != nil {
			self.log.WithError(err).Error("Error in run()")
		}
	}()

	return
}

// Receives data from the input channel and saves in the database
func (self *Store) run() (err error) {
	// Used to ensure data isn't stuck in Syncer for too long
	ticker := time.NewTicker(self.config.StoreMaxTimeInQueue)

	var pendingInteractions []*model.Interaction
	var lastTransactionBlockHeight int64

	insert := func() {
		if len(pendingInteractions) == 0 {
			return
		}
		self.log.WithField("length", len(pendingInteractions)).Debug("Insert batch of interactions")
		err = self.DB.WithContext(self.Ctx).
			// Clauses(clause.OnConflict{
			// 	// Columns:   cols,
			// 	// DoUpdates: clause.AssignmentColumns(colsNames),
			// 	DoNothing: true,
			// }).
			CreateInBatches(pendingInteractions, len(pendingInteractions)).
			Error
		if err != nil {
			self.log.WithError(err).Error("Failed to insert Interactions")
			// Close to avoid holes in inserted data
			// self.Stop()
			// TODO: Maybe it's possible to retry on some errors

			// TODO: Send an email
			// FIXME: Retry
		}

		// FIXME: This isn't the right value. This should be the last downloaded transaction
		err = self.setLastTransactionBlockHeight(self.Ctx, lastTransactionBlockHeight)
		if err != nil {
			// FIXME: Retry
			self.log.WithError(err).Error("Failed to update last transaction block height")
		}

		// Reset buffer index
		pendingInteractions = nil

		// Prolong time to forced insert
		ticker.Reset(self.config.StoreMaxTimeInQueue)
	}
	for {
		select {
		case <-self.stopChannel:
			// Stop was requested, close the input channel
			// Won't accept new data, but will process pending
			ticker.Stop()
			close(self.input)

		case payload, ok := <-self.input:
			if !ok {
				// The only way input channel is closed is that the Store is stopping
				// There will be no more data, insert everything there is and quit.
				insert()

				// NOTE: This (and panic()) is the only way to quit run()
				return
			}

			self.log.WithField("height", payload.BlockHeight).WithField("num_interactions", len(pendingInteractions)).Info("Got block")
			lastTransactionBlockHeight = payload.BlockHeight

			pendingInteractions = append(pendingInteractions, payload.Interactions...)

			if len(pendingInteractions) >= self.config.StoreBatchSize {
				insert()
			}

		case <-ticker.C:
			if len(pendingInteractions) > 0 {
				self.log.Debug("Batch timed out, trigger insert")
				insert()
			}
		}
	}
}

func (self *Store) Save(ctx context.Context, payload *Payload) (err error) {
	if self.isStopping.Load() {
		self.log.Error("Tried to store interaction after Store got stopped. Something's wrong in stopping order.")
		return
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case self.input <- payload:
	}
	return
}

func (self *Store) StopWait() {
	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if self.isStopping.CompareAndSwap(false, true) {
		self.stopChannel <- true
	}

	// Store's context will be cancelled only after processing all pending messages
	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, some data may have been not stored")
	case <-self.Ctx.Done():
		self.log.Info("Store stopped")
	}

}

func (self *Store) GetLastTransactionBlockHeight(ctx context.Context) (out int64, err error) {
	var state model.State
	err = self.DB.WithContext(ctx).First(&state).Error
	return state.LastTransactionBlockHeight, err
}

func (self *Store) setLastTransactionBlockHeight(ctx context.Context, value int64) (err error) {
	state := model.State{Id: 1}
	return self.DB.WithContext(ctx).Model(&state).Update("last_transaction_block_height", value).Error
}
