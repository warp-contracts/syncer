package forward

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
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

			// TODO: Neglect messages that are too big
			select {
			case <-self.Ctx.Done():
			case out <- &model.InteractionNotification{
				ContractTxId: data.Interaction.ContractId,
				Test:         false,
				Source:       "warp-gw",
				Interaction:  data.Interaction.Interaction,
			}:
			}

			return nil
		})
}
