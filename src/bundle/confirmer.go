package bundle

import (
	"database/sql"
	"syncer/src/utils/config"
	"syncer/src/utils/listener"
	"syncer/src/utils/model"
	"syncer/src/utils/monitoring"
	"syncer/src/utils/task"

	"github.com/jackc/pgtype"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

// Periodically saves the confirmation state of the bundlet interactions
// This is done to prevent flooding database with bundle_items state updates
type Confirmer struct {
	*task.SinkTask[*Confirmation]
	monitor        monitoring.Monitor
	db             *gorm.DB
	networkMonitor *listener.NetworkMonitor
}

type Confirmation struct {
	InteractionID int
	BundlerTxID   string
	Response      pgtype.JSONB
}

func NewConfirmer(config *config.Config) (self *Confirmer) {
	self = new(Confirmer)

	self.SinkTask = task.NewSinkTask[*Confirmation](config, "confirmer").
		WithBatchSize(config.Bundler.ConfirmerBatchSize).
		WithOnFlush(config.Bundler.ConfirmerInterval, self.save).
		WithBackoff(config.Bundler.ConfirmerBackoffMaxElapsedTime, config.Bundler.ConfirmerBackoffMaxInterval)

	return
}

func (self *Confirmer) WithDB(db *gorm.DB) *Confirmer {
	self.db = db
	return self
}

func (self *Confirmer) WithInputChannel(input chan *Confirmation) *Confirmer {
	self.SinkTask.WithInputChannel(input)
	return self
}

func (self *Confirmer) WithMonitor(monitor monitoring.Monitor) *Confirmer {
	self.monitor = monitor
	return self
}

func (self *Confirmer) WithNetworkMonitor(v *listener.NetworkMonitor) *Confirmer {
	self.networkMonitor = v
	return self
}

func (self *Confirmer) save(confirmations []*Confirmation) error {
	if len(confirmations) == 0 {
		// Nothing to save
		return nil
	}

	self.Log.WithField("len", len(confirmations)).Info("Saving confirmations to DB")

	// Sort confirmations by interaction ID to minimize deadlocks
	slices.SortFunc(confirmations, func(a, b *Confirmation) bool {
		return a.InteractionID < b.InteractionID
	})

	// Network manager updates this value
	// NOTE: This can potentially block if NetworkMonitor can't get the first height
	currentBlockHeight := self.networkMonitor.GetLastNetworkInfo().Height

	// Uses one transaction to do all the updates
	// NOTE: It still uses many requests to the database,
	// it should be possible to combine updates into batches, but it's not a priority for now.
	err := self.db.Transaction(func(tx *gorm.DB) (err error) {
		for _, confirmation := range confirmations {
			err := tx.Model(&model.BundleItem{
				InteractionID: confirmation.InteractionID,
			}).
				Where("state = ?", model.BundleStateUploading).
				Updates(model.BundleItem{
					State:          model.BundleStateUploaded,
					BlockHeight:    sql.NullInt64{Int64: currentBlockHeight, Valid: true},
					BundlrResponse: confirmation.Response,
				}).
				Error
			if err != nil {
				return err
			}

			err = tx.Model(&model.Interaction{}).
				Where("id = ?", confirmation.InteractionID).
				Update("bundler_tx_id", confirmation.BundlerTxID).
				Error
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		self.Log.WithError(err).Error("Failed to save bundle items, retrying...")

		// Update monitoring
		self.monitor.GetReport().Bundler.Errors.ConfirmationsSavedToDbError.Inc()
		return err
	}

	// Update monitoring
	self.monitor.GetReport().Bundler.State.ConfirmationsSavedToDb.Inc()

	return nil

}
