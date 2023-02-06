package bundle

import (
	"context"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"

	"github.com/gammazero/workerpool"
	"gorm.io/gorm"
)

type InteractionMonitor struct {
	*task.Task
	db *gorm.DB

	// Pool of workers that actually do the check.
	// It's possible to run multiple requests in parallel.
	// We're limiting the number of parallel requests with the number of workers.
	workers *workerpool.WorkerPool
}

// Main class that orchestrates main syncer functionalities
func NewInteractionMonitor(config *config.Config, db *gorm.DB) (self *InteractionMonitor, err error) {
	self = new(InteractionMonitor)
	self.db = db

	self.Task = task.NewTask(config, "interaction-monitor").
		WithSubtaskFunc(self.monitorInteractions)

	// Worker pool for downloading interactions in parallel
	self.workers = workerpool.New(self.Config.InteractionManagerMaxParallelQueries)

	return
}

func (self *InteractionMonitor) monitorInteractions() error {
	ticker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-self.StopChannel:
			self.Log.Debug("Stop monitoring interactions")

			return nil
		case <-ticker.C:
			self.Log.Debug("Tick")
			if self.workers.WaitingQueueSize() > 1 {
				self.Log.Debug("Too many pending interaction checks")
				continue
			}
			// self.workers.Submit(self.check)
		}
	}
}

// ALTER TABLE interactions ADD COLUMN state interaction_state NOT NULL DEFAULT 'PENDING'::interaction_state;
func (self *InteractionMonitor) check() {
	self.Log.Debug("Getting interactions to bundle")

	ctx, cancel := context.WithTimeout(self.Ctx, self.Config.InteractionManagerTimeout)
	defer cancel()

	var interactions []model.Interaction

	err := self.db.Model(interactions).
		WithContext(ctx).
		Raw(`WITH rows AS (
			SELECT id
			FROM interactions
			WHERE state = 'PENDING'::interaction_state
			ORDER BY id
			LIMIT ?
		  )
		  UPDATE interactions
		  SET state = 'UPLOADING'::interaction_state
		  WHERE id IN (SELECT id FROM rows)
		  RETURNING *`, 10).
		Scan(&interactions).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get interactions")
	}

	self.Log.WithField("length", len(interactions)).Debug("Out")

}
