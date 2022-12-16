package poller

import (
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syncer/src/utils/model"

	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Repository handles saving data to the database in na robust way.
// - groups incoming Interactions into batches,
// - ensures data isn't stuck even if a batch isn't big enough
type Repository struct {
	Ctx    context.Context
	cancel context.CancelFunc

	config *config.Config
	log    *logrus.Entry
	input  chan *model.Interaction
	DB     *gorm.DB
}

func NewRepository(ctx context.Context, config *config.Config) (self *Repository, err error) {
	self = new(Repository)
	self.log = logger.NewSublogger("repository")
	self.config = config

	// Connection to the database
	self.DB, err = model.NewConnection(ctx, self.config)
	if err != nil {
		return
	}

	// Incoming interactions channel
	self.input = make(chan *model.Interaction)

	// Global context for closing everything
	self.Ctx, self.cancel = context.WithCancel(ctx)

	return
}

func (self *Repository) Start() {
	go func() {
		defer func() {
			var err error
			if p := recover(); p != nil {
				switch p := p.(type) {
				case error:
					err = p
				default:
					err = fmt.Errorf("%s", p)
				}
				self.log.WithError(err).Error("Panic in Repository. Stopping.")
				self.Stop()
				panic(p)
			}
		}()
		err := self.run()
		if err != nil {
			self.log.WithError(err).Error("Error in run()")
		}
	}()
}

func (self *Repository) Save(ctx context.Context, interaction *model.Interaction) (err error) {
	// self.log.Debug("Save interaction")
	// defer self.log.Debug("Interaction queued")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case self.input <- interaction:
	}
	return
}

// Receives data from the input channel and saves in the database
func (self *Repository) run() (err error) {
	// Used to ensure data isn't stuck in Syncer for too long
	ticker := time.NewTicker(self.config.RepositoryMaxTimeInQueue)

	// Fixed size buffer
	interactions := make([]*model.Interaction, self.config.RepositoryBatchSize)
	idx := 0

	insert := func() {
		if idx == 0 {
			return
		}
		self.log.WithField("length", idx).Debug("Insert batch of interactions")
		err = self.DB.WithContext(self.Ctx).
			// Clauses(clause.OnConflict{
			// 	// Columns:   cols,
			// 	// DoUpdates: clause.AssignmentColumns(colsNames),
			// 	DoNothing: true,
			// }).
			CreateInBatches(interactions[:idx], self.config.RepositorySubBatchSize).
			Error
		if err != nil {
			self.log.WithError(err).Error("Failed to insert Interactions")
			// Close to avoid holes in inserted data
			// self.Stop()
			// TODO: Maybe it's possible to retry on some errors
		}

		// Reset buffer index
		idx = 0

		// Prolong forced insert
		ticker.Reset(self.config.RepositoryMaxTimeInQueue)
	}
	for {
		select {
		case <-self.Ctx.Done():
			// Listener is closing
			ticker.Stop()
			return
		case interaction, ok := <-self.input:
			if !ok {
				// Channel closed, close the repository
				self.cancel()
				return
			}

			interactions[idx] = interaction
			idx += 1

			if idx < self.config.RepositoryBatchSize {
				// Buffer isn't full yet, don't
				continue
			}

			insert()
		case <-ticker.C:
			if idx != 0 {
				self.log.Debug("Batch timed out, trigger insert")
			}
			insert()
		}
	}
}

func (self *Repository) Stop() {
	self.log.Info("Stopping Repository...")
	defer self.log.Info("Repository stopped")

	close(self.input)
	self.cancel()

}
