package sync

import (
	"syncer/src/utils/config"
	"syncer/src/utils/logger"

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

	stopChannel chan bool

	server  *Server
	monitor *Monitor
}

// Main class that orchestrates main syncer functionalities
// Setups listening and storing interactions
func NewController(config *config.Config) (self *Controller, err error) {
	self = new(Controller)
	self.log = logger.NewSublogger("controller")
	self.config = config

	// Controller's context, it's valid as long the controller is running.
	self.Ctx, self.cancel = context.WithCancel(context.Background())

	// Internal channel for closing the underlying goroutine
	self.stopChannel = make(chan bool, 1)

	self.monitor = NewMonitor()

	// REST server API, healthchecks
	self.server, err = NewServer(config, self.monitor)
	if err != nil {
		return
	}

	return
}

func (self *Controller) Start() {
	// REST API server
	self.server.Start()

	go func() {
		defer func() {
			// run() finished, so it's time to cancel Controller's context
			// NOTE: This should be the only place self.Ctx is cancelled
			self.cancel()

			var err error
			if p := recover(); p != nil {
				switch p := p.(type) {
				case error:
					err = p
				default:
					err = fmt.Errorf("%s", p)
				}
				self.log.WithError(err).Error("Panic in sync. Stopping.")
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
	store := NewStore(self.config, self.monitor)
	err = store.Start()
	if err != nil {
		return
	}
	defer store.StopWait()

	// Get the last stored block height
	startHeight, err := store.GetLastTransactionBlockHeight(self.Ctx)
	if err != nil {
		return
	}

	// Listening for arweave transactions
	listener, err := NewListener(self.config)
	if err != nil {
		return
	}

	listener.Start(startHeight + 1)
	defer listener.StopWait()

	for {
		select {
		case <-self.stopChannel:
			self.log.Info("Controller is stopping")
			listener.Stop()
		case payload, ok := <-listener.Payloads:
			if !ok {
				// Listener stopped
				return
			}

			err = store.Save(self.Ctx, payload)
			if err != nil {
				self.log.WithError(err).Error("Failed to store payload")
				// Log the error, but neglect it
			}
		}
	}
}

func (self *Controller) StopWait() {
	self.log.Info("Stopping Controller...")

	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(self.Ctx, 5*time.Second)
	defer cancel()

	// Trigger stopping
	self.stopChannel <- true

	// Wait for the pending messages to be sent
	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, some data may have been not sent")
	case <-self.Ctx.Done():
		self.log.Info("Controller stopped")
	}

	// Stop REST API server
	self.server.StopWait()
}
