package relay

import (
	"context"
	"errors"
	"fmt"
	"time"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/tool"
)

// Fills the last arweave block in the Payload
// Upon startup get info from the Sequencer, but later it just caches the info
type LastArweaveBlockProvider struct {
	*task.Task

	monitor monitoring.Monitor
	client  *rpchttp.HTTP
	decoder *Decoder

	input  <-chan *Payload
	Output chan *Payload

	// Value taken from the API and updated upon MsgArweaveBlock
	lastArweaveBlockHeight int64
}

// Converts Arweave transactions into Warp's contracts
func NewLastArweaveBlockProvider(config *config.Config) (self *LastArweaveBlockProvider) {
	self = new(LastArweaveBlockProvider)

	self.Output = make(chan *Payload)

	self.lastArweaveBlockHeight = -1

	self.Task = task.NewTask(config, "last_arweave_block_provider").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *LastArweaveBlockProvider) WithMonitor(monitor monitoring.Monitor) *LastArweaveBlockProvider {
	self.monitor = monitor
	return self
}

func (self *LastArweaveBlockProvider) WithInputChannel(v <-chan *Payload) *LastArweaveBlockProvider {
	self.input = v
	return self
}

func (self *LastArweaveBlockProvider) WithClient(client *rpchttp.HTTP) *LastArweaveBlockProvider {
	self.client = client
	return self
}

func (self *LastArweaveBlockProvider) WithDecoder(decoder *Decoder) *LastArweaveBlockProvider {
	self.decoder = decoder
	return self
}

func (self *LastArweaveBlockProvider) getLastBlockHeight(payload *Payload) (out int64, err error) {
	ctx, cancel := context.WithTimeout(self.Ctx, time.Minute)
	defer cancel()

	query := fmt.Sprintf("tx.height <= %d AND message.action='/sequencer.sequencer.MsgArweaveBlock'", payload.SequencerBlockHeight)

	results, err := self.client.TxSearch(ctx, query, false /*prove*/, nil /*page*/, nil /*per page*/, "" /*order by*/)
	if err != nil {
		return
	}

	if results.TotalCount == 0 && len(results.Txs) == 0 {
		err = errors.New("no arweave blocks found, relayer started before any block was mined")
		return
	}

	tx, err := self.decoder.Decode(results.Txs[0].Tx)
	if err != nil {
		return
	}

	msg, ok := tx.GetMsgs()[0].(*sequencertypes.MsgArweaveBlock)
	if !ok {
		err = errors.New("failed to decode arweave block message")
		return
	}

	out = int64(msg.BlockInfo.Height)

	self.Log.WithField("last_arweave_block_height", msg.BlockInfo.Height).Info("Got last arweave block from sequencer")

	return
}

func (self *LastArweaveBlockProvider) fill(payload *Payload) (err error) {
	// Update cache
	for _, arweaveBlock := range payload.ArweaveBlocks {
		self.lastArweaveBlockHeight = tool.Max(self.lastArweaveBlockHeight, int64(arweaveBlock.Message.BlockInfo.Height))
	}

	// Use updated cache
	if self.lastArweaveBlockHeight > 0 {
		// self.Log.WithField("last_arweave_block_height", self.lastArweaveBlockHeight).Debug("Use last arweave block ")
		payload.LastArweaveBlockHeight = self.lastArweaveBlockHeight
		return
	}

	// Request last arweave block for the given sequencer height
	self.lastArweaveBlockHeight, err = self.getLastBlockHeight(payload)
	return
}

func (self *LastArweaveBlockProvider) run() (err error) {
	for payload := range self.input {
		err = self.fill(payload)
		if err != nil {
			if self.IsStopping.Load() {
				// Neglect, we're stopping anyway
				return nil
			}

			self.Log.WithField("sequencer_height", payload.SequencerBlockHeight).WithError(err).Error("Failed to fill last arweave block height")

			// Stop everything
			// We can't neglect parsing errors
			panic(err)
		}

		select {
		case <-self.Ctx.Done():
			err = errors.New("task closing")
			return
		case self.Output <- payload:
		}
	}

	return nil
}
