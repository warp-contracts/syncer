package interact

import (
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
	"github.com/warp-contracts/syncer/src/utils/task"
	"gorm.io/gorm"
)

// Periodically gets the current network height from warp's GW and confirms bundle is FINALIZED
type Checker struct {
	*task.Task

	monitor monitoring.Monitor
	db      *gorm.DB
	// Data items that should be in the database after couple of seconds
	input <-chan *Payload
}

// Receives bundles that can be checked in bundlr
func NewChecker(config *config.Config) (self *Checker) {
	self = new(Checker)

	self.Task = task.NewTask(config, "checker").
		WithSubtaskFunc(self.run)

	return
}

func (self *Checker) WithDb(db *gorm.DB) *Checker {
	self.db = db
	return self
}

func (self *Checker) WithInputChannel(input <-chan *Payload) *Checker {
	self.input = input
	return self
}

func (self *Checker) WithMonitor(monitor monitoring.Monitor) *Checker {
	self.monitor = monitor
	return self
}

func (self *Checker) check(payload *Payload) error {
	// Check if the data item is in the database
	var interaction model.Interaction

	for i := 0; i < 100; i++ {
		err := self.db.Table(model.TableInteraction).
			First(&interaction, "interaction_id = ?", payload.DataItem.Id.Base64()).
			Error
		if err == gorm.ErrRecordNotFound {
			self.Log.WithField("id", payload.DataItem.Id.Base64()).Debug("Waiting for data item to be saved to database")
			select {
			case <-self.Ctx.Done():
				return nil
			case <-time.After(10 * time.Second):
			}
			continue
		}
		if err != nil {
			self.Log.WithError(err).Error("Failed to get data item from database")
			return err
		}

		// Check some fields, just in case
		input, ok := payload.DataItem.Tags.Get(smartweave.TagInput)
		if !ok || interaction.Input != input {
			self.Log.Warn("Input doesn't match")
		}

		// Get the delay
		delay := payload.Timestamp.UnixMilli() - interaction.SyncTimestamp.Int
		self.monitor.GetReport().Interactor.State.CheckerDelay.Store(delay)

		self.Log.WithField("delay", delay).Info("Data item detected in database")

		return nil
	}

	self.Log.WithField("id", payload.DataItem.Id.Base64()).Error("Data item not found in database")
	return nil
}

func (self *Checker) run() error {
	for payload := range self.input {

		// Wait for couple of seconds to let the data item be saved to the database
		select {
		case <-self.Ctx.Done():
			return nil
		case <-time.After(10 * time.Second):
		}

		// Check if it's there
		err := self.check(payload)
		if err != nil {
			return err
		}
	}

	return nil
}
