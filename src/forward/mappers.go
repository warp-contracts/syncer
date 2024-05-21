package forward

import (
	"fmt"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	publisher "github.com/warp-contracts/syncer/src/utils/publisher"
	"github.com/warp-contracts/syncer/src/utils/task"
)

func redisMapper(config *config.Config) (self *task.Mapper[*Payload, *model.InteractionNotification]) {
	return task.NewMapper[*Payload, *model.InteractionNotification](config, "map-redis-notification").
		WithWorkerPool(1, config.Forwarder.FetcherBatchSize).
		WithProcessFunc(func(data *Payload, out chan *model.InteractionNotification) (err error) {
			// Neglect empty messages
			if data.Interaction == nil {
				return nil
			}

			self.Log.WithField("contract_id", data.Interaction.ContractId).Trace("Publishing interaction to Redis")

			interactionStr, err := data.Interaction.Interaction.MarshalJSON()
			if err != nil {
				self.Log.WithField("contract_id", data.Interaction.ContractId).Warn("Failed to marshal interaction")
				return err
			}

			// TODO: Neglect messages that are too big
			select {
			case <-self.Ctx.Done():
			case out <- &model.InteractionNotification{
				ContractTxId: data.Interaction.ContractId,
				Test:         false,
				Source:       "warp-gw",
				Interaction:  string(interactionStr),
				SrcTxId:      data.SrcTxId,
			}:
			}

			return nil
		})
}

func appSyncMapper(config *config.Config, channelName string, forContract bool) (self *task.Mapper[*Payload, *publisher.AppSyncPayload[*model.InteractionNotification]]) {
	return task.NewMapper[*Payload, *publisher.AppSyncPayload[*model.InteractionNotification]](config, "map-appsync-notification").
		WithWorkerPool(1, config.Contract.StoreBatchSize).
		WithProcessFunc(func(data *Payload, out chan *publisher.AppSyncPayload[*model.InteractionNotification]) (err error) {
			channelName := channelName

			// Neglect empty messages
			if data.Interaction == nil {
				return nil
			}

			self.Log.WithField("contract_id", data.Interaction.ContractId).Trace("Publishing interaction to AppSync")

			interactionStr, err := data.Interaction.Interaction.MarshalJSON()
			if err != nil {
				self.Log.WithField("contract_id", data.Interaction.ContractId).Warn("Failed to marshal interaction")
				return err
			}

			if forContract {
				channelName = fmt.Sprintf("%s/%s", channelName, data.Interaction.ContractId)
			}

			select {
			case <-self.Ctx.Done():
			case out <- &publisher.AppSyncPayload[*model.InteractionNotification]{In: &model.InteractionNotification{
				ContractTxId: data.Interaction.ContractId,
				Test:         false,
				Source:       "warp-gw",
				Interaction:  string(interactionStr),
				SrcTxId:      data.SrcTxId,
			},
				ChannelName: channelName,
			}:
			}

			return nil
		})
}
