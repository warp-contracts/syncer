package forward

import (
	"encoding/json"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/streamer"
	"syncer/src/utils/task"

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
		// Get the last storeserverd block height from the database
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

		if sequencerState.FinishedBlockHeight > forwarderState.FinishedBlockHeight {
			select {
			case <-self.Ctx.Done():
				return nil
			case self.Output <- sequencerState.FinishedBlockHeight:
			}
		}

		self.currentHeight = forwarderState.FinishedBlockHeight

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
				self.Log.Info("Syncer changed its finished height")
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

			// Update height that we are currently at
			self.currentHeight = state.FinishedBlockHeight

			// Update metrics
			self.monitor.GetReport().Forwarder.State.CurrentHeight.Inc()

			select {
			case <-self.Ctx.Done():
				return nil
			case self.Output <- state.FinishedBlockHeight:
			}

		}
	}
}
