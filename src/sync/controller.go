package sync

import (
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syncer/src/utils/model"

	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	Ctx    context.Context
	cancel context.CancelFunc

	config *config.Config
	log    *logrus.Entry
}

// Main class that orchestrates main syncer functionalities
// Setups listening and storing interactions
func NewController(ctx context.Context, config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.log = logger.NewSublogger("controller")
	self.config = config

	// Global context for closing everything
	self.Ctx, self.cancel = context.WithCancel(ctx)

	return
}

func (self *Controller) Start() {
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
				self.log.WithError(err).Error("Panic in sync. Stopping.")
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

func (self *Controller) run() (err error) {
	// Stores interactions
	store, err := NewStore(self.Ctx, self.config)
	if err != nil {
		return
	}
	store.Start()
	defer store.Stop()

	// Get the last stored block height
	startHeight, err := model.LastBlockHeight(self.Ctx, store.DB)
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
		case <-store.Ctx.Done():
			self.log.Warn("Store stopped")
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
			err = store.Save(self.Ctx, interaction)
			if err != nil {
				self.log.WithError(err).Error("Failed to store interaction")
				// Log the error, but neglect it
			}
		}
	}
}

func (self *Controller) Stop() {
	self.log.Info("Stopping sync...")
	defer self.log.Info("Sync stopped")

	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(self.Ctx, 30*time.Second)
	defer cancel()

	// Wait for the pending messages to be sent
	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, some data may have been not sent")
	case <-self.Ctx.Done():
		self.log.Error("Force quit sync")
	}
	self.cancel()
}
