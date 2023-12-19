package interact

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/sequencer"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/warp"
)

// Periodically gets the current network height from warp's GW and confirms bundle is FINALIZED
type Generator struct {
	*task.Task

	monitor monitoring.Monitor
	Output  chan *Payload
	signer  bundlr.Signer

	previousId      []byte
	sequencerClient *sequencer.Client
}

type Input struct {
	Timestamp time.Time `json:"t"`
}

// For every network height, fetches unfinished bundles
func NewGenerator(config *config.Config) (self *Generator) {
	self = new(Generator)

	self.Output = make(chan *Payload)

	// Ethereum signer because it's faster
	var err error
	self.signer, err = bundlr.NewEthereumSigner(config.Interactor.GeneratorEthereumKey)
	if err != nil {
		panic(err)
	}

	self.Task = task.NewTask(config, "generator").
		WithPeriodicSubtaskFunc(config.Interactor.GeneratorInterval, self.generate)

	return
}

func (self *Generator) WithMonitor(monitor monitoring.Monitor) *Generator {
	self.monitor = monitor
	return self
}

func (self *Generator) WithClient(v *sequencer.Client) *Generator {
	self.sequencerClient = v
	return self
}

func (self *Generator) generateDataItem(nonce uint64, timestamp time.Time) (out *bundlr.BundleItem, err error) {
	// Draw size
	out = new(bundlr.BundleItem)
	out.SignatureType = bundlr.SignatureTypeEthereum

	// Random anchor
	out.Anchor = make([]byte, 32)
	if len(self.previousId) != 0 {
		copy(out.Anchor, self.previousId)
	} else {
		var n int
		n, err = rand.Read(out.Anchor)
		if n != 32 {
			err = errors.New("failed to generate random anchor")
			return
		}
		if err != nil {
			return
		}
	}

	// Input json
	input, err := json.Marshal(Input{Timestamp: timestamp})
	if err != nil {
		return
	}

	// Nonce tag
	out.Tags = append(out.Tags,
		bundlr.Tag{Name: warp.TagSequencerNonce, Value: strconv.FormatUint(nonce, 10)},
		bundlr.Tag{Name: "Contract", Value: self.Config.Interactor.GeneratorContractId},
		bundlr.Tag{Name: smartweave.TagAppName, Value: smartweave.TagAppNameValue},
		bundlr.Tag{Name: smartweave.TagInputFormat, Value: smartweave.TagInputFormatTagValue},
		bundlr.Tag{Name: smartweave.TagInput, Value: string(input)},
	)

	// Sign
	err = out.Sign(self.signer)
	if err != nil {
		return
	}

	return
}

func (self *Generator) generate() error {
	self.Log.Debug("Generating bundle item")

	// Get nonce
	r, resp, err := self.sequencerClient.GetNonce(self.Ctx, self.signer.GetType(), base64.RawURLEncoding.EncodeToString(self.signer.GetOwner()))
	if err != nil {
		self.Log.WithError(err).Error("Failed to get nonce")
		return nil
	}

	if !resp.IsSuccess() {
		self.Log.WithField("resp", string(resp.Body())).Error("Response is not success")
		return nil
	}

	self.Log.WithError(err).WithField("nonce", r.Nonce).Info("Got nonce")

	timestamp := time.Now()
	item, err := self.generateDataItem(r.Nonce, timestamp)
	if err != nil {
		return err
	}

	select {
	case <-self.Ctx.Done():
		return nil
	case self.Output <- &Payload{
		DataItem:  item,
		Timestamp: timestamp,
	}:
		return nil
	}
}
