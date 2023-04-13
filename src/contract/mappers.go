package contract

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/model"
	"syncer/src/utils/task"
	"time"
)

func redisMapper(config *config.Config) (self *task.Mapper[*ContractData, *model.ContractNotification]) {
	return task.NewMapper[*ContractData, *model.ContractNotification](config, "map-redis-notification").
		WithWorkerPool(1, config.Contract.StoreBatchSize).
		WithProcessFunc(func(data *ContractData, out chan *model.ContractNotification) (err error) {
			// Neglect messages that are too big
			if len(data.Contract.InitState.Bytes) > self.Config.Contract.PublisherMaxMessageSize {
				self.Log.WithField("contract_id", data.Contract.ContractId).
					WithField("len", len(data.Contract.InitState.Bytes)).
					Warn("Init state too big for notifications, skipping")
				return err
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

			return nil
		})
}

func appSyncMapper(config *config.Config) (self *task.Mapper[*ContractData, *model.AppSyncContractNotification]) {
	return task.NewMapper[*ContractData, *model.AppSyncContractNotification](config, "map-redis-notification").
		WithWorkerPool(1, config.Contract.StoreBatchSize).
		WithProcessFunc(func(data *ContractData, out chan *model.AppSyncContractNotification) (err error) {
			select {
			case <-self.Ctx.Done():
			case out <- &model.AppSyncContractNotification{
				ContractTxId: data.Contract.ContractId,
				// PPE: hmm, for L1 contracts source should be 'arweave'
				// and in general it should be taken from already parsed Contract
				Source:         "warp-external",
				BlockHeight:    data.Contract.BlockHeight,
				BlockTimestamp: data.Contract.BlockTimestamp,
				Creator:        data.Contract.Owner.String,
				Type:           data.Contract.Type.String,

				// PPE: this should be taken from contract.sync_timestamp
				SyncTimestamp: time.Now().Unix(),
			}:
			}

			return nil
		})
}
