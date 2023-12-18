package evolve

import (
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store struct {
	*task.Hole[*model.ContractSource]

	db      *gorm.DB
	monitor monitoring.Monitor
}

func NewStore(config *config.Config) (self *Store) {
	self = new(Store)

	self.Hole = task.NewHole[*model.ContractSource](config, "store").
		WithBatchSize(config.Evolver.StoreBatchSize).
		WithOnFlush(config.Evolver.StoreInterval, self.flush).
		WithBackoff(config.Evolver.StoreBackoffMaxElapsedTime, config.Evolver.StoreBackoffMaxInterval)

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

func (self *Store) WithMonitor(monitor monitoring.Monitor) *Store {
	self.monitor = monitor
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
		CreateInBatches(&contractSources, self.Config.Evolver.StoreBatchSize).
		Error
	if err != nil {
		self.Log.WithError(err).Error("Failed to insert contract sources, retrying...")

		// Update monitoring
		self.monitor.GetReport().Evolver.Errors.StoreDbError.Inc()
		return err
	}

	// Update monitoring
	self.monitor.GetReport().Evolver.State.StoreSourcesSaved.Add(uint64(len(contractSources)))

	return nil
}
