package load

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/task"
	"time"
)

// Generates fake transactions
type Generator struct {
	*task.Task

	Output chan *arweave.Transaction
}

func NewGenerator(config *config.Config) (self *Generator) {
	self = new(Generator)

	self.Output = make(chan *arweave.Transaction)

	self.Task = task.NewTask(config, "generator").
		WithPeriodicSubtaskFunc(2000*time.Millisecond, self.runPeriodically)

	return
}

func (self *Generator) runPeriodically() error {
	tx := self.fakeTransaction()

	select {
	case <-self.StopChannel:
	case self.Output <- tx:
	}
	return nil
}

func (self *Generator) fakeTransaction() *arweave.Transaction {
	return &arweave.Transaction{}
}
