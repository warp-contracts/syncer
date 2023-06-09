package forward

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"
	"github.com/warp-contracts/syncer/src/utils/tool"
	"gorm.io/gorm"
)

// Gets L1 (Arweave) interactions from the DB in batches
// Fills in last_sort_key for each interaction before emiting to the output channel
type ArweaveFetcher struct {
	*task.Task
	db *gorm.DB

	monitor monitoring.Monitor

	input  chan uint64
	Output chan *Payload
}

func NewArweaveFetcher(config *config.Config) (self *ArweaveFetcher) {
	self = new(ArweaveFetcher)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "arweave-fetcher").
		WithSubtaskFunc(self.run)

	return
}

func (self *ArweaveFetcher) WithDB(db *gorm.DB) *ArweaveFetcher {
	self.db = db
	return self
}

func (self *ArweaveFetcher) WithMonitor(monitor monitoring.Monitor) *ArweaveFetcher {
	self.monitor = monitor
	return self
}

func (self *ArweaveFetcher) WithInputChannel(input chan uint64) *ArweaveFetcher {
	self.input = input
	return self
}

func (self *ArweaveFetcher) run() (err error) {
	for height := range self.input {
		self.Log.WithField("height", height).Debug("New height to fetch")
		// Cache Contract id -> interaction sort key
		lastSortKeys := make(map[string]string)

		isFirstBatch := true

		// Fetch interactions in batches, offset is the batch number
		for offset := 0; ; offset++ {
			// Fetch interactions in batches
			var interactions []*model.Interaction

			err = self.db.WithContext(self.Ctx).
				Transaction(func(tx *gorm.DB) (err error) {
					// Get a batch of L1 interactions
					err = self.db.Table(model.TableInteraction).
						Where("block_height = ?", height).
						Where("source=?", "arweave").
						Limit(self.Config.Forwarder.FetcherBatchSize).
						Offset(offset * self.Config.Forwarder.FetcherBatchSize).

						// FIXME: -----------------> IS THIS ORDERING CORRECT? <-----------------
						// This is the order DRE gets L1 interactions
						Order("sort_key ASC").
						Find(&interactions).
						Error
					if err != nil {
						return
					}
					if len(interactions) == 0 {
						return
					}

					// Update last_sort_key for each interaction in the database
					// As a optimization lastSortKeys are cached in memory
					lastSortKeys, err = self.updateLastSortKey(tx, interactions, height, lastSortKeys)
					if err != nil {
						return
					}

					// Update sync height
					return self.updateSyncedHeight(tx, height)
				})
			if err != nil {
				self.Log.WithError(err).Error("Failed to fetch interactions from DB")
				return
			}

			if len(interactions) == 0 && offset != 0 {
				// Edge case: num of interactions is a multiple of batch size
				payload := &Payload{First: false, Last: true, Interaction: nil}
				self.Output <- payload
			} else {
				isLastBatch := len(interactions) < self.Config.Forwarder.FetcherBatchSize
				for i, interaction := range interactions {
					payload := &Payload{
						First:       isFirstBatch && i == 0,
						Last:        isLastBatch && i == len(interactions)-1,
						Interaction: interaction,
					}

					// NOTE: Quit only when the whole batch is processed
					// That's why we're not waiting for closing of this task
					self.Output <- payload
				}
			}

			isFirstBatch = false
		}
	}
	return
}

func (self *ArweaveFetcher) updateSyncedHeight(tx *gorm.DB, height uint64) (err error) {
	var state model.State
	err = tx.WithContext(self.Ctx).
		Where("name = ?", model.SyncedComponentForwarder).
		First(&state).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to get state")
		return err
	}

	if state.FinishedBlockHeight < height {
		state.FinishedBlockHeight = height

		err = tx.Model(&model.State{
			Name: model.SyncedComponentForwarder,
		}).
			Updates(model.State{
				FinishedBlockHeight: height,
			}).
			Error
		if err != nil {
			self.Log.WithError(err).Error("Failed to update stmonitorate after last block")
			self.monitor.GetReport().Contractor.Errors.DbLastTransactionBlockHeight.Inc()
			return err
		}
	}
	return
}

func (self *ArweaveFetcher) updateLastSortKey(tx *gorm.DB, interactions []*model.Interaction, height uint64, lastSortKeys map[string]string) (out map[string]string, err error) {
	// Get contract ids of fetched interactions
	// Omit those that are already in the lastSortKeys map
	newContractIds := self.getNewContractIds(interactions, lastSortKeys)

	// Get last sort key for each contract
	newLastSortKeys, err := self.getLastSortKeys(tx, newContractIds, height)
	if err != nil {
		return
	}

	// Merge new LSK into the existing map
	out = tool.AppendMap(lastSortKeys, newLastSortKeys)

	// Fill in last sort key for each interaction
	for _, interaction := range interactions {
		interaction.LastSortKey = out[interaction.ContractId]
		out[interaction.ContractId] = interaction.SortKey
	}

	// Update last sort key for each contract
	for _, interaction := range interactions {
		err = tx.Model(interaction).
			Update("last_sort_key", interaction.LastSortKey).
			Error
		if err != nil {
			return
		}
	}
	return
}

func (self *ArweaveFetcher) getNewContractIds(interactions []*model.Interaction, lastSortKeys map[string]string) (out []string) {
	contractIds := make(map[string]struct{})
	for _, interaction := range interactions {
		_, ok := lastSortKeys[interaction.ContractId]
		if ok {
			// There's already a sort key for this contract id
			continue
		}
		contractIds[interaction.ContractId] = struct{}{}
	}

	for contractId := range contractIds {
		out = append(out, contractId)
	}

	return
}

func (self *ArweaveFetcher) getLastSortKeys(tx *gorm.DB, contractIds []string, height uint64) (out map[string]string, err error) {
	out = make(map[string]string)

	// TODO: Receive a dedicated structure, not Interaction
	var interactions []*model.Interaction
	err = tx.Table(model.TableInteraction).
		Select("contract_id, MAX(sort_key) AS sort_key").
		Where("contract_id IN ?", contractIds).
		Where("block_height < ?", height).
		Group("contract_id").
		Find(&interactions).
		Error
	if err != nil {
		return
	}

	for _, interaction := range interactions {
		out[interaction.ContractId] = interaction.SortKey
	}

	return
}
