package poller

import (
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syncer/src/utils/model"

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
				self.log.WithError(err).Error("Panic in poller. Stopping.")
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

func (self *Poller) run() (err error) {
	// Stores interactions
	repository, err := NewRepository(self.Ctx, self.config)
	if err != nil {
		return
	}
	repository.Start()
	defer repository.Stop()

	// Get the last stored block height
	startHeight, err := model.LastBlockHeight(self.Ctx, repository.DB)
	if err != nil {
		return
	}

	// Listening for arweave transactions
	listener, err := NewListener(self.Ctx, self.config, startHeight+1)
	if err != nil {
		return
	}

	listener.Start()
	defer listener.Stop()

	for {
		select {
		case <-repository.Ctx.Done():
			self.log.Warn("Repository stopped")
			self.cancel()
			return
		case <-listener.Ctx.Done():
			self.log.Warn("Listener stopped")
			self.cancel()
			return
		case <-self.Ctx.Done():
			self.log.Warn("Syncer is stopping")
			return
		case interaction, ok := <-listener.Interactions:
			if !ok {
				// Channel closed, close the listener
				self.cancel()
				return
			}
			err = repository.Save(self.Ctx, interaction)
			if err != nil {
				self.log.WithError(err).Error("Failed to store interaction")
				// Log the error, but neglect it
			}
		}
	}
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
