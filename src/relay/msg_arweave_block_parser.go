package relay

import (
	"errors"

	proto "github.com/cosmos/gogoproto/proto"
	sequencertypes "github.com/warp-contracts/sequencer/x/sequencer/types"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
)

// Parses sequencer.sequencer.MsgArweaveBlock
type MsgArweaveBlockParser struct {
	*task.Task

	monitor monitoring.Monitor

	input  <-chan *Payload
	Output chan *Payload
}

// Converts Arweave transactions into Warp's contracts
func NewMsgArweaveBlockParser(config *config.Config) (self *MsgArweaveBlockParser) {
	self = new(MsgArweaveBlockParser)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "msg_arweave_block_parser").
		WithSubtaskFunc(self.run).
		WithOnAfterStop(func() {
			close(self.Output)
		})

	return
}

func (self *MsgArweaveBlockParser) WithMonitor(monitor monitoring.Monitor) *MsgArweaveBlockParser {
	self.monitor = monitor
	return self
}

func (self *MsgArweaveBlockParser) WithInputChannel(v <-chan *Payload) *MsgArweaveBlockParser {
	self.input = v
	return self
}

func (self *MsgArweaveBlockParser) parse(payload *Payload) (out *Payload, err error) {
	var ok bool
	for _, msg := range payload.Messages {
		if proto.MessageName(msg) != "sequencer.sequencer.MsgArweaveBlock" {
			continue
		}

		arweaveBlock := new(ArweaveBlock)
		arweaveBlock.Message, ok = msg.(*sequencertypes.MsgArweaveBlock)
		if !ok {
			err = errors.New("failed to cast MsgArweaveBlock")
			return
		}

		payload.ArweaveBlocks = append(payload.ArweaveBlocks, arweaveBlock)
	}

	out = payload

	return
}

func (self *MsgArweaveBlockParser) run() (err error) {
	var payload *Payload
	for inPayload := range self.input {
		payload, err = self.parse(inPayload)
		if err != nil {
			if self.IsStopping.Load() {
				// Neglect, we're stopping anyway
				return nil
			}

			self.Log.WithField("sequencer_height", payload.SequencerBlockHeight).WithError(err).Error("Failed to parse msg arweave block messages")

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
