package load

import (
	"encoding/base64"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/task"
	"syncer/src/utils/tool"
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
		WithPeriodicSubtaskFunc(10*time.Millisecond, self.runPeriodically)

	return
}

func (self *Generator) runPeriodically() error {
	// self.Log.Info("Generated tx")
	tx := self.fakeTransaction()

	select {
	case <-self.StopChannel:
	case self.Output <- tx:
	}
	return nil
}

func (self *Generator) fakeTransaction() *arweave.Transaction {
	return &arweave.Transaction{
		Signature: arweave.Base64String("fake"),
		Owner:     arweave.Base64String("fake"),
		Tags: []arweave.Tag{
			{Name: arweave.Base64String("bundler-tests"), Value: arweave.Base64String("true")},
			{Name: arweave.Base64String(tool.RandomString(43)), Value: arweave.Base64String(tool.RandomString(43))},
			{Name: arweave.Base64String(tool.RandomString(43)), Value: arweave.Base64String(tool.RandomString(43))},
			{Name: arweave.Base64String(tool.RandomString(43)), Value: arweave.Base64String(tool.RandomString(43))},
			{Name: arweave.Base64String(tool.RandomString(43)), Value: arweave.Base64String(tool.RandomString(10))},
		},
		Data: base64.RawURLEncoding.EncodeToString([]byte(tool.RandomString(10000))),
		ID:   tool.RandomString(43),
	}
}
