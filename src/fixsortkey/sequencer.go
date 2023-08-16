package fixsortkey

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Produces network info in a sequential order, but only from the specified range
type Sequencer struct {
	*task.Task
	db *gorm.DB

	// Current Syncer's block height
	Output chan uint64

	// Current, broadcasted height
	startHeight uint64
	stopHeight  uint64
}

func NewSequencer(config *config.Config) (self *Sequencer) {
	self = new(Sequencer)

	self.Output = make(chan uint64)

	self.Task = task.NewTask(config, "sequencer").
		WithSubtaskFunc(self.run)

	return
}

func (self *Sequencer) WithDB(db *gorm.DB) *Sequencer {
	self.db = db
	return self
}

func (self *Sequencer) WithStart(height uint64) *Sequencer {
	self.startHeight = height
	return self
}

func (self *Sequencer) WithStop(height uint64) *Sequencer {
	self.stopHeight = height
	return self
}

func (self *Sequencer) run() (err error) {
	for i := self.startHeight; i <= self.stopHeight; i++ {
		self.Log.WithField("progress", float64(i-self.startHeight)/float64(self.stopHeight-self.startHeight)*100.0).WithField("h", i).Info("Emitted block height")
		select {
		case <-self.Ctx.Done():
			return
		case self.Output <- i:
		}
	}

	self.Log.Info("Finished broadcasting block heights")

	// Wait till the context is done
	<-self.Ctx.Done()

	return
}
