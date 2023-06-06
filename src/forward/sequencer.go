package forward

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/streamer"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Produces current syncer's height
type Sequencer struct {
	*task.Task
	db *gorm.DB

	streamer *streamer.Streamer
	monitor  monitoring.Monitor

	// Current Syncer's block height
	Output chan uint64

	// Current, broadcasted height
	currentHeight uint64
}

func NewSequencer(config *config.Config) (self *Sequencer) {
	self = new(Sequencer)

	self.Output = make(chan uint64)

	self.streamer = streamer.NewStreamer(config, "sequence-sync-state").
		WithNotificationChannelName("sync_state_sequencer").
		WithCapacity(10)

	self.Task = task.NewTask(config, "sequencer").
		// Live source of interactions that need to be bundled
		WithSubtask(self.streamer.Task).
		// Interactions that somehow wasn't sent through the notification channel. Probably because of a restart.
		WithSubtaskFunc(self.run)

	return
}

func (self *Sequencer) WithDB(db *gorm.DB) *Sequencer {
	self.db = db
	return self
}

func (self *Sequencer) WithMonitor(monitor monitoring.Monitor) *Sequencer {
	self.monitor = monitor
	return self
}

func (self *Sequencer) WithInitStartHeight(db *gorm.DB) *Sequencer {
	self.Task = self.Task.WithOnBeforeStart(func() (err error) {
		// Get the last block height from the database
		var sequencerState, forwarderState model.State

		err = db.WithContext(self.Ctx).
			Transaction(func(tx *gorm.DB) error {
				err = tx.WithContext(self.Ctx).
					Table(model.TableState).
					Find(&forwarderState, model.SyncedComponentForwarder).
					Error
				if err != nil {
					return err
				}
				return tx.WithContext(self.Ctx).
					Table(model.TableState).
					Find(&sequencerState, model.SyncedComponentSequencer).
					Error
			})
		if err != nil {
			self.Log.WithError(err).Error("Failed to sync state for forwarder and sequencer")
			return
		}

		self.Log.WithField("sequencer_finished_height", sequencerState.FinishedBlockHeight).
			WithField("forwarder_finished_height", forwarderState.FinishedBlockHeight).
			Info("Downloaded sync state")

		// Emit height change one by one
		self.currentHeight = forwarderState.FinishedBlockHeight
		err = self.emit(sequencerState.FinishedBlockHeight)
		if err != nil {
			return
		}

		return nil
	})
	return self
}

func (self *Sequencer) run() (err error) {
	for {
		select {
		case <-self.Ctx.Done():
			self.Log.Debug("Stop passing sync state")
			return nil
		case msg, ok := <-self.streamer.Output:
			if !ok {
				self.Log.Error("Streamer closed, can't receive sequencer's state changes!")
				return nil
			}

			var state model.State
			err = json.Unmarshal([]byte(msg), &state)
			if err != nil {
				self.Log.WithError(err).Error("Failed to unmarshal sequencer sync state")
				return
			}

			if state.FinishedBlockHeight <= self.currentHeight {
				// There was no change, neglect
				continue
			}

			// Emit height change one by one
			err = self.emit(state.FinishedBlockHeight)
			if err != nil {
				return
			}
		}
	}
}

func (self *Sequencer) emit(newHeight uint64) (err error) {
	for newHeight > self.currentHeight {
		time.Sleep(self.Config.Forwarder.HeightDelay)
		// Update height that we are currently at
		self.currentHeight += 1
		select {
		case <-self.Ctx.Done():
			return errors.New("Sequencer stopped")
		case self.Output <- self.currentHeight:
		}

		// Update metrics
		self.monitor.GetReport().Forwarder.State.CurrentHeight.Inc()
	}
	return
}
