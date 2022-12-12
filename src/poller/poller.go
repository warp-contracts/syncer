package poller

import (
	"syncer/src/utils/config"
	"syncer/src/utils/logger"

	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type Poller struct {
	Ctx    context.Context
	cancel context.CancelFunc

	config *config.Config
	log    *logrus.Entry
}

func NewPoller(ctx context.Context, config *config.Config) (self *Poller, err error) {
	self = new(Poller)
	self.log = logger.NewSublogger("poller")
	self.config = config

	// Global context for closing everything
	self.Ctx, self.cancel = context.WithCancel(ctx)

	return
}

func (self *Poller) Start() {
	go self.run()
}

// Started every time a connection is established
func (self *Poller) run() {
	defer func() {
		var err error
		if p := recover(); p != nil {
			switch p := p.(type) {
			case error:
				err = p
			default:
				err = fmt.Errorf("%s", p)
			}
			self.log.WithError(err).Error("Panic in poller. Stopping.")
			self.Stop()
			panic(p)
		}
	}()
}

func (self *Poller) Stop() {
	self.log.Info("Stopping poller...")
	defer self.log.Info("Poller stopped")

	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(self.Ctx, 30*time.Second)
	defer cancel()

	// Wait for the pending messages to be sent
	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, some data may have been not sent")
	case <-self.Ctx.Done():
		self.log.Error("Force quit poller")
	}
	self.cancel()
}
