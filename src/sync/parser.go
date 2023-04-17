package sync

import (
	"sync"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"
	"syncer/src/utils/warp"
)

// Gets contract's source and init state
type Parser struct {
	*task.Task

	monitor           monitoring.Monitor
	interactionParser *warp.InteractionParser

	input  chan *listener.Payload
	Output chan *Payload
}

// Converts Arweave transactions into Warp's contracts
func NewParser(config *config.Config) (self *Parser) {
	self = new(Parser)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "contract-Parser").
		WithSubtaskFunc(self.run).
		WithWorkerPool(config.ListenerNumWorkers, config.ListenerWorkerQueueSize).
		WithOnAfterStop(func() {
			close(self.Output)
		}).
		WithOnBeforeStart(func() error {
			// Converting Arweave transactions to interactions
			var err error
			self.interactionParser, err = warp.NewInteractionParser(config)
			return err
		})

	return
}

func (self *Parser) WithMonitor(monitor monitoring.Monitor) *Parser {
	self.monitor = monitor
	return self
}

func (self *Parser) WithInputChannel(v chan *listener.Payload) *Parser {
	self.input = v
	return self
}

func (self *Parser) run() error {
	// Each payload has a slice of transactions
	for payload := range self.input {

		interactions, err := self.parseAll(payload)
		if err != nil {
			return err
		}

		select {
		case <-self.Ctx.Done():
			return nil
		case self.Output <- &Payload{
			BlockHeight:  uint64(payload.BlockHeight),
			BlockHash:    payload.BlockHash,
			Interactions: interactions,
		}:
		}
	}

	return nil
}

func (self *Parser) parseAll(payload *listener.Payload) (out []*model.Interaction, err error) {
	var wg sync.WaitGroup
	wg.Add(len(payload.Transactions))
	var mtx sync.Mutex

	// Fill int
	out = make([]*model.Interaction, 0, len(payload.Transactions))
	for _, tx := range payload.Transactions {
		tx := tx
		self.SubmitToWorker(func() {
			// Parse transactions into interaction
			interaction, err := self.interactionParser.Parse(tx, payload.BlockHeight, payload.BlockHash, payload.BlockTimestamp)
			if err != nil {
				self.monitor.GetReport().Syncer.State.FailedInteractionParsing.Inc()
				self.Log.WithField("tx_id", tx.ID).Warn("Failed to parse interaction from tx, neglecting")
				goto done
			}

			mtx.Lock()
			out = append(out, interaction)
			mtx.Unlock()

		done:
			wg.Done()
		})
	}

	wg.Wait()
	return
}
