package evolve

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)
 
type Store struct {
	*task.Hole[*model.ContractSource]

	db      *gorm.DB
	input   chan *model.ContractSource
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)
 
	self.Hole = task.NewHole[*model.ContractSource](config, "evolve-store").
		WithBatchSize(config.Evolve.StoreBatchSize).
		WithOnFlush(config.Evolve.StoreInterval, self.flush).
		WithBackoff(config.Evolve.StoreBackoffMaxElapsedTime, config.Evolve.StoreBackoffMaxInterval)

	return
}

func (self *Store) WithInputChannel(input chan *model.ContractSource) *Store {
	self.Hole.WithInputChannel(input)
	return self
}

func (self *Store) WithDB(db *gorm.DB) *Store {
	self.db = db
	return self
}

func (self *Store) flush(contractSources []*model.ContractSource) (err error) {
	if len(contractSources) == 0 {
		return nil
	}

	self.Log.WithField("len", len(contractSources)).Debug("-> Saving evolved contract sources to DB")
	defer self.Log.WithField("len", len(contractSources)).Debug("<- Saving evolved contract sources to DB")

	err = self.db.WithContext(self.Ctx).
			Table(model.TableContractSource).
			Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "src_tx_id"}},
				DoNothing: true,
			}).
			CreateInBatches(&contractSources, self.Config.Sender.StoreBatchSize).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to insert contract sources, retrying...")
		return err
	}

	return nil
}
