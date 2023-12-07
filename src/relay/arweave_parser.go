package relay

import (
	"runtime"
	"sync"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
)

// Parses Arweave transactions into Warp's interactions
// Passes other payloads through
type ArweaveParser struct {
	*task.Task

	monitor           monitoring.Monitor
	interactionParser *warp.InteractionParser

	input  chan *Payload
	Output chan *Payload
}

// Converts Arweave transactions into Warp's interactions
func NewArweaveParser(config *config.Config) (self *ArweaveParser) {
	self = new(ArweaveParser)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "arweave-parser").
		WithSubtaskFunc(self.run).
		WithWorkerPool(runtime.NumCPU(), 1000).
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

func (self *ArweaveParser) WithMonitor(monitor monitoring.Monitor) *ArweaveParser {
	self.monitor = monitor
	return self
}

func (self *ArweaveParser) WithInputChannel(v chan *Payload) *ArweaveParser {
	self.input = v
	return self
}

func (self *ArweaveParser) run() error {
	for payload := range self.input {
		for i, arweaveBlock := range payload.ArweaveBlocks {
			var err error
			payload.ArweaveBlocks[i].Interactions, err = self.parseAll(arweaveBlock)
			if err != nil {
				if self.IsStopping.Load() {
					// Neglect those transactions, we're stopping anyway
					return nil
				}

				// We don't allow parsing errors because all those transactions were already parsed in sequecer
				// Neglecting them here would cause loosing data

				self.Log.WithField("sequencer_block_height", payload.SequencerBlockHeight).
					WithField("arweave_block_height", arweaveBlock.Message.BlockInfo.Height).
					WithError(err).
					Error("Persistent error. Failed to parse some Arweave transactions into interactions.")

				// Stop everything
				// We can't neglect a parsing error
				panic(err)
			}

			// FIXME: create a bundle item with the order of arweave blocks
			self.Log.WithField("height", arweaveBlock.Message.BlockInfo.Height).
				WithField("hash", arweaveBlock.Message.BlockInfo.Hash).
				WithField("len", len(payload.ArweaveBlocks[i].Interactions)).
				Debug("Parsed interactions")
		}

		select {
		case <-self.Ctx.Done():
			return nil
		case self.Output <- payload:
		}
	}

	return nil
}

func (self *ArweaveParser) parseAll(arweaveBlock *ArweaveBlock) (out []*model.Interaction, err error) {
	if len(arweaveBlock.Transactions) == 0 {
		// Skip empty blocks
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(arweaveBlock.Transactions))
	var mtx sync.Mutex

	// Fill interactions, order doesn't matter
	out = make([]*model.Interaction, 0, len(arweaveBlock.Transactions))
	for i, tx := range arweaveBlock.Transactions {
		tx := tx
		i := i
		self.SubmitToWorker(func() {
			info := arweaveBlock.Message.Transactions[i]
			// Parse transactions into interaction
			interaction, errParse := self.interactionParser.Parse(tx,
				arweaveBlock.Block.Height,
				arweaveBlock.Block.IndepHash,
				arweaveBlock.Block.Timestamp,
				info.Random,
				info.Transaction.SortKey,
				info.PrevSortKey,
			)
			mtx.Lock()
			defer mtx.Unlock()

			if errParse != nil {
				self.Log.WithError(err).WithField("tx", tx.String()).WithField("tx_id", tx.ID.Base64()).Warn("Failed to parse interaction from tx")
				self.monitor.GetReport().Relayer.Errors.PersistentArweaveFailedParsing.Inc()

				if err == nil {
					// Don't overwrite previous error
					err = errParse
				}

				goto done
			}

			out = append(out, interaction)

		done:
			wg.Done()
		})
	}

	wg.Wait()

	return
}
