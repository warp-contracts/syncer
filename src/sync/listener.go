package sync

import (
	"encoding/json"
	"sync/atomic"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syncer/src/utils/model"
	"syncer/src/utils/smartweave"

	"context"
	"fmt"
	"time"

	"github.com/everFinance/arsyncer"
	"github.com/everFinance/goar/types"
	"github.com/everFinance/goar/utils"
	"github.com/sirupsen/logrus"
)

type Listener struct {
	Ctx    context.Context
	cancel context.CancelFunc

	config *config.Config
	log    *logrus.Entry

	Interactions chan *model.Interaction

	stopChannel chan bool
	isStopping  *atomic.Bool
}

// Listens for changes
func NewListener(config *config.Config) (self *Listener) {
	self = new(Listener)
	self.log = logger.NewSublogger("listener")
	self.config = config
	self.Interactions = make(chan *model.Interaction, config.ListenerQueueSize)

	// Listener context, active as long as there's anything running in Listener
	self.Ctx, self.cancel = context.WithCancel(context.Background())
	self.Ctx = common.SetConfig(self.Ctx, config)

	// Internal channel for closing the underlying goroutine
	self.stopChannel = make(chan bool, 1)

	// Variable used for avoiding stopping Listener two times upon panics/errors
	self.isStopping = &atomic.Bool{}
	return
}

func (self *Listener) Start(startHeight int64) {
	go func() {
		defer func() {
			// run() finished, so it's time to cancel Listener's context
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
				self.log.WithError(err).Error("Panic in Listener. Stopping.")

				// NOTE: Panics in listener are suppressed
				// because syncer is using panics for reporting errors...
				// panic(p)
			}
		}()
		self.run(startHeight)
	}()
}

func (self *Listener) run(startHeight int64) {
	// Setup arsyncer notifications
	syncer := arsyncer.New(
		startHeight,
		arsyncer.FilterParams{
			Tags: []types.Tag{
				{Name: "App-Name", Value: "SmartWeaveAction"},
			},
		},
		self.config.ArNodeUrl,
		self.config.ArConcurrentConnections,
		self.config.ArStableDistance,
		arsyncer.SubscribeTypeTx)
	syncer.Run()

	for {
		select {
		case <-self.stopChannel:
			// Stop was requested, trigger closing connections
			syncer.Close()
		case block, ok := <-syncer.SubscribeTxCh():
			if !ok {
				// Listener is closing and closing channels was requested.
				// All pending messages got processed. Close the outgoing channel, there won't be any more data.
				close(self.Interactions)

				// NOTE: This (and panic()) is the only way to quit run()
				return
			}
			for _, tx := range block {
				interaction, err := self.parse(&tx)
				if err != nil {
					self.log.WithField("tx_id", tx.ID).Warn("Failed to parse transaction")
					continue
				}
				self.Interactions <- interaction
			}
		}
	}
}

func (self *Listener) parse(tx *arsyncer.SubscribeTx) (out *model.Interaction, err error) {
	decodedTags, err := utils.TagsDecode(tx.Tags)
	if err != nil {
		return
	}

	out = &model.Interaction{
		InteractionId:      tx.ID,
		BlockHeight:        tx.BlockHeight,
		BlockId:            tx.BlockId,
		ConfirmationStatus: "not_processed",
	}

	// Fill data from tags
	for _, t := range decodedTags {
		switch t.Name {
		case "Contract":
			out.ContractId = t.Value
		case "Input":
			out.Input = t.Value

			var parsedInput map[string]interface{}
			err = json.Unmarshal([]byte(out.Input), &parsedInput)
			if err != nil {
				self.log.Error("Failed to parse function in input")
				return
			}

			val, ok := parsedInput["function"]
			if !ok {
				out.Function, ok = val.(string)
				if !ok {
					self.log.Error("Function field isn't a string")
					return
				}
			}
		}

		if out.ContractId != "" && out.Input != "" {
			break
		}
	}

	swInteraction := smartweave.Interaction{
		Id: tx.ID,
		Owner: smartweave.Owner{
			Address: tx.Owner,
		},
		Recipient: tx.Target,
		Tags:      decodedTags,
		Block: smartweave.Block{
			Height:    tx.BlockHeight,
			Id:        tx.BlockId,
			Timestamp: tx.BlockTimestamp,
		},
		Fee: smartweave.Amount{
			Winston: tx.Reward,
		},
		Quantity: smartweave.Amount{
			Winston: tx.Quantity,
		},
	}

	swInteractionJson, err := json.Marshal(swInteraction)
	if err != nil {
		self.log.Error("Failed to marshal interaction")
		return
	}
	out.Interaction = string(swInteractionJson)

	return
}

func (self *Listener) Stop() {
	self.log.Info("Stopping Listener...")
	if self.isStopping.CompareAndSwap(false, true) {
		// Listener wasn't stopping before, trigger
		self.stopChannel <- true
	}
}

// Stops listener and waits for everything to finish
func (self *Listener) StopSync() {
	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	self.Stop()

	// Wait for the pending messages to be sent
	select {
	case <-self.Ctx.Done():
		self.log.Info("Listener finished")
	case <-ctx.Done():
		self.log.Error("Timeout reached, failed to finish listening")
	}
}
