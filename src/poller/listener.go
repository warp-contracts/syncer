package poller

import (
	"encoding/json"
	"syncer/src/utils/config"
	"syncer/src/utils/db"
	"syncer/src/utils/logger"
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
	syncer *arsyncer.Syncer

	Interactions chan *db.Interaction
}

// Listens for changes
func NewListener(ctx context.Context, config *config.Config, startHeight int64) (self *Listener, err error) {
	self = new(Listener)
	self.log = logger.NewSublogger("listener")
	self.config = config
	self.Interactions = make(chan *db.Interaction, config.ListenerQueueSize)
	// Global context for closing everything
	self.Ctx, self.cancel = context.WithCancel(ctx)

	// Setup arsyncer notifications
	// Arsyncer setups many connections and each of them may put one
	self.syncer = arsyncer.New(
		startHeight,
		arsyncer.FilterParams{
			Tags: []types.Tag{
				{Name: "App-Name", Value: "SmartWeaveAction"},
			},
		},
		config.ArNodeUrl,
		config.ArConcurrentConnections,
		config.ArStableDistance,
		arsyncer.SubscribeTypeTx)

	return
}

func (self *Listener) Start() {
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
				self.log.WithError(err).Error("Panic in Listener. Stopping.")
				self.Stop()
				panic(p)
			}
		}()
		self.syncer.Run()
		self.receive()
	}()
}

func (self *Listener) receive() {
	for {
		select {
		case <-self.Ctx.Done():
			// Listener is closing
			return
		case transactions, ok := <-self.syncer.SubscribeTxCh():
			if !ok {
				// Channel closed, close the listener
				self.cancel()
				return
			}
			for _, tx := range transactions {
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

func (self *Listener) parse(tx *arsyncer.SubscribeTx) (out *db.Interaction, err error) {
	decodedTags, err := utils.TagsDecode(tx.Tags)
	if err != nil {
		return
	}

	out = &db.Interaction{
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
	defer self.log.Info("Listener stopped")

	// Wait for at most 30s before force-closing
	ctx, cancel := context.WithTimeout(self.Ctx, 30*time.Second)
	defer cancel()

	self.syncer.Close()

	// Wait for the pending messages to be sent
	select {
	case <-ctx.Done():
		self.log.Error("Timeout reached, failed to finish listening")
	case <-self.Ctx.Done():
		self.log.Error("Force quit Listener")
	}
	self.cancel()
}
