package contract

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
)

// Maps batches of ContractData to payloads for the Publisher
// Filters out messages that are too big
type Mapper struct {
	*task.Mapper[[]*ContractData, *model.ContractNotification]
}

func NewMapper(config *config.Config) (self *Mapper) {
	self = new(Mapper)

	self.Mapper = task.NewMapper[[]*ContractData, *model.ContractNotification](config, "map-notification").
		WithWorkerPool(1, config.Contract.StoreBatchSize).
		WithProcessFunc(self.process)

	return
}

func (self *Mapper) WithInputChannel(v chan []*ContractData) *Mapper {
	self.Mapper = self.Mapper.WithInputChannel(v)
	return self
}

func (self *Mapper) process(in []*ContractData, out chan *model.ContractNotification) (err error) {
	for _, data := range in {
		// Neglect messages that are too big
		if len(data.Contract.InitState.Bytes) > self.Config.Contract.PublisherMaxMessageSize {
			self.Log.WithField("contract_id", data.Contract.ContractId).
				WithField("len", len(data.Contract.InitState.Bytes)).
				Warn("Init state too big for notifications, skipping")
			continue
		}
		select {
		case <-self.Ctx.Done():
		case out <- &model.ContractNotification{
			ContractTxId: data.Contract.ContractId,
			Test:         false,
			Source:       "warp-gw", // FIXME: Should this be changed to another name?
			InitialState: data.Contract.InitState,
			Tags:         []arweave.Tag{}, // Empty array as in the GW
		}:
		}
	}

	return nil
}
