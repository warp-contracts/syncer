package sync

// import (
// 	"sync/atomic"
// 	"syncer/src/utils/common"
// 	"syncer/src/utils/config"
// 	"syncer/src/utils/logger"
// 	"syncer/src/utils/model"
// 	"syncer/src/utils/warp"

// 	"context"
// 	"fmt"
// 	"time"

// 	"github.com/everFinance/arsyncer"
// 	"github.com/everFinance/goar/types"
// 	"github.com/sirupsen/logrus"
// )

// type Listener struct {
// 	Ctx    context.Context
// 	cancel context.CancelFunc

// 	config *config.Config
// 	log    *logrus.Entry

// 	Payloads chan *Payload

// 	stopChannel chan bool
// 	isStopping  *atomic.Bool

// 	interactionParser *warp.InteractionParser
// }

// // Listens for changes
// func NewListener(config *config.Config) (self *Listener, err error) {
// 	self = new(Listener)
// 	self.log = logger.NewSublogger("listener")
// 	self.config = config
// 	self.Payloads = make(chan *Payload, config.ListenerQueueSize)

// 	// Listener context, active as long as there's anything running in Listener
// 	self.Ctx, self.cancel = context.WithCancel(context.Background())
// 	self.Ctx = common.SetConfig(self.Ctx, config)

// 	// Internal channel for closing the underlying goroutine
// 	self.stopChannel = make(chan bool, 1)

// 	// Variable used for avoiding stopping Listener two times upon panics/errors
// 	self.isStopping = &atomic.Bool{}

// 	// Converting Arweave transactions to interactions
// 	self.interactionParser, err = warp.NewInteractionParser(config)
// 	if err != nil {
// 		return
// 	}
// 	return
// }

// func (self *Listener) Start(startHeight int64) {
// 	go func() {
// 		defer func() {
// 			// run() finished, so it's time to cancel Listener's context
// 			// NOTE: This should be the only place self.Ctx is cancelled
// 			self.cancel()

// 			var err error
// 			if p := recover(); p != nil {
// 				switch p := p.(type) {
// 				case error:
// 					err = p
// 				default:
// 					err = fmt.Errorf("%s", p)
// 				}
// 				self.log.WithError(err).Error("Panic in Listener. Stopping.")

// 				// NOTE: Panics in listener are suppressed
// 				// because syncer is using panics for reporting errors...
// 				// panic(p)
// 			}
// 		}()
// 		self.run(startHeight)
// 	}()
// }

// func (self *Listener) run(startHeight int64) {
// 	// Setup arsyncer notifications
// 	syncer := arsyncer.New(
// 		startHeight,
// 		arsyncer.FilterParams{
// 			Tags: []types.Tag{
// 				{Name: "App-Name", Value: "SmartWeaveAction"},
// 			},
// 		},
// 		self.config.ArNodeUrl,
// 		self.config.ArConcurrentConnections,
// 		self.config.ArStableDistance,
// 		arsyncer.SubscribeTypeTx)
// 	syncer.Run()

// 	for {
// 		select {
// 		case <-self.stopChannel:
// 			// Stop was requested, trigger closing connections
// 			syncer.Close()
// 		case block, ok := <-syncer.SubscribeTxCh():
// 			if !ok {
// 				// Listener is closing and closing channels was requested.
// 				// All pending messages got processed. Close the outgoing channel, there won't be any more data.
// 				close(self.Payloads)

// 				// NOTE: This (and panic()) is the only way to quit run()
// 				return
// 			}

// 			// TODO: Divide array if it's too big
// 			payload := &Payload{
// 				BlockHeight:  block[0].BlockHeight,
// 				Interactions: make([]*model.Interaction, len(block)),
// 			}
// 			var err error
// 			for idx, tx := range block {
// 				payload.Interactions[idx], err = self.interactionParser.Parse(&tx)
// 				if err != nil {
// 					self.log.WithField("tx_id", tx.ID).Warn("Failed to parse transaction")
// 					continue
// 				}
// 			}

// 			self.Payloads <- payload
// 		}
// 	}
// }

// func (self *Listener) Stop() {
// 	self.log.Info("Stopping Listener...")
// 	if self.isStopping.CompareAndSwap(false, true) {
// 		// Listener wasn't stopping before, trigger
// 		self.stopChannel <- true
// 	}
// }

// // Stops listener and waits for everything to finish
// func (self *Listener) StopWait() {
// 	// Wait for at most 30s before force-closing
// 	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
// 	defer cancel()

// 	self.Stop()

// 	// Wait for the pending messages to be sent
// 	select {
// 	case <-self.Ctx.Done():
// 		self.log.Info("Listener finished")
// 	case <-ctx.Done():
// 		self.log.Error("Timeout reached, failed to finish listening")
// 	}
// }
