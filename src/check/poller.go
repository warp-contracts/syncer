package check

import (
	"context"
	"fmt"
	"time"

	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/listener"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/monitoring"
	"github.com/warp-contracts/syncer/src/utils/task"

	"gorm.io/gorm"
)

// Periodically gets the current network height from warp's GW and confirms bundle is FINALIZED
type Poller struct {
	*task.Task
	db      *gorm.DB
	monitor monitoring.Monitor

	networkMonitor *listener.NetworkMonitor
	Output         chan *Payload
}

// For every network height, fetches unfinished bundles
func NewPoller(config *config.Config) (self *Poller) {
	self = new(Poller)

	self.Output = make(chan *Payload)

	self.Task = task.NewTask(config, "poller").
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleCheck(model.BundlingServiceIrys)).
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleCheck(model.BundlingServiceTurbo)).
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleRetrying(model.BundlingServiceIrys)).
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleRetrying(model.BundlingServiceTurbo)).
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleDataItemCheck(model.BundlingServiceIrys)).
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleDataItemCheck(model.BundlingServiceTurbo)).
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleDataItemRetrying(model.BundlingServiceIrys)).
		WithRepeatedSubtaskFunc(config.Checker.PollerInterval, self.handleDataItemRetrying(model.BundlingServiceTurbo))

	return
}

func (self *Poller) WithDB(db *gorm.DB) *Poller {
	self.db = db
	return self
}

func (self *Poller) WithNetworkMonitor(v *listener.NetworkMonitor) *Poller {
	self.networkMonitor = v
	return self
}

func (self *Poller) WithMonitor(monitor monitoring.Monitor) *Poller {
	self.monitor = monitor
	return self
}

func (self *Poller) handleCheck(bundlingService model.BundlingService) task.RepeatedSubtaskFunc {
	return func() (repeat bool, err error) {

		// Get the current network height
		networkInfo := self.networkMonitor.GetLastNetworkInfo()
		minHeightToCheck := networkInfo.Height - self.Config.Checker.MinConfirmationBlocks

		// Interrupts longer queries
		ctx, cancel := context.WithTimeout(self.Ctx, 5*time.Minute)
		defer cancel()

		// Preallocate the slices
		ids := make([]int, 0, self.Config.Checker.MaxBundlesPerRun)
		interactions := make([]model.Interaction, 0, self.Config.Checker.MaxBundlesPerRun)

		err = self.db.WithContext(ctx).
			Transaction(func(tx *gorm.DB) error {
				// Select bundles that will get checked
				err = tx.Raw(`UPDATE bundle_items
			SET state = 'CHECKING', updated_at = NOW()
					WHERE interaction_id IN (
						SELECT interaction_id 
						FROM bundle_items 
						WHERE state = 'UPLOADED' AND block_height < ? AND service = ?
						ORDER BY block_height ASC, interaction_id ASC
						LIMIT ?
						FOR UPDATE SKIP LOCKED)
						RETURNING interaction_id`, minHeightToCheck, bundlingService.String(), self.Config.Checker.MaxBundlesPerRun).
					Scan(&ids).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get bundles to check")
					return err
				}

				if len(ids) == 0 {
					return nil
				}

				// Get the data from interactions table
				err := tx.Table(model.TableInteraction).
					Select("id", "bundler_tx_id").
					Where("id IN ?", ids).
					Where("bundler_tx_id IS NOT NULL").
					Where("bundler_tx_id <> ''").
					Scan(&interactions).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get interactions for checking")
					return err
				}
				return nil
			})

		if err != nil {
			self.Log.WithError(err).Error("Failed to get data for checking")
			return
		}

		if len(interactions) > 0 {
			self.Log.WithField("len", len(interactions)).Debug("Polled interactions for checking")
		}

		for _, interaction := range interactions {
			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &Payload{
				InteractionId: interaction.ID,
				BundlerTxId:   interaction.BundlerTxId,
				Service:       bundlingService,
				Table:         model.TableBundleItem,
			}:
			}
		}

		// Update monitoring
		self.monitor.GetReport().Checker.State.BundlesTakenFromDb.Add(uint64(len(interactions)))

		// If we got the maximum number of elements, we need to repeat the query
		if len(ids) == self.Config.Checker.MaxBundlesPerRun {
			repeat = true
		}
		return

	}
}

func (self *Poller) handleRetrying(bundlingService model.BundlingService) task.RepeatedSubtaskFunc {
	return func() (repeat bool, err error) {
		// Interrupts longer queries
		ctx, cancel := context.WithTimeout(self.Ctx, 5*time.Minute)
		defer cancel()

		// Preallocate the slices
		ids := make([]int, 0, self.Config.Checker.MaxBundlesPerRun)
		interactions := make([]model.Interaction, 0, self.Config.Checker.MaxBundlesPerRun)

		err = self.db.WithContext(ctx).
			Transaction(func(tx *gorm.DB) error {
				// Select bundles that will get checked
				err = tx.Raw(`UPDATE bundle_items
						SET updated_at = NOW()
						WHERE interaction_id IN (
							SELECT interaction_id 
							FROM bundle_items 
							WHERE state = 'CHECKING'  AND service = ? AND updated_at < NOW() - ?::interval 
							ORDER BY block_height ASC, interaction_id ASC
							LIMIT ?
							FOR UPDATE SKIP LOCKED)
						RETURNING interaction_id`,
					bundlingService.String(),
					fmt.Sprintf("%d seconds", int((self.Config.Checker.PollerRetryCheckAfter.Seconds()))),
					self.Config.Checker.MaxBundlesPerRun).
					Scan(&ids).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get bundles to re-check")
					return err
				}

				if len(ids) == 0 {
					return nil
				}

				// Get the data from interactions table
				err := tx.Table(model.TableInteraction).
					Select("id", "bundler_tx_id").
					Where("id IN ?", ids).
					Where("bundler_tx_id IS NOT NULL").
					Where("bundler_tx_id <> ''").
					Scan(&interactions).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get interactions for re-checking")
					return err
				}
				return nil
			})
		if err != nil {
			self.Log.WithError(err).Error("Failed to get data for checking")
			return
		}

		if len(interactions) > 0 {
			self.Log.WithField("len", len(interactions)).Debug("Polled interactions for re-checking")
		}

		for _, interaction := range interactions {
			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &Payload{
				InteractionId: interaction.ID,
				BundlerTxId:   interaction.BundlerTxId,
				Service:       bundlingService,
				Table:         model.TableBundleItem,
			}:
			}
		}

		// Update monitoring
		self.monitor.GetReport().Checker.State.BundlesTakenFromDb.Add(uint64(len(interactions)))

		// If we got the maximum number of elements, we need to repeat the query
		if len(ids) == self.Config.Checker.MaxBundlesPerRun {
			repeat = true
		}
		return
	}
}

func (self *Poller) handleDataItemCheck(bundlingService model.BundlingService) task.RepeatedSubtaskFunc {
	return func() (repeat bool, err error) {
		// Get the current network height
		networkInfo := self.networkMonitor.GetLastNetworkInfo()
		minHeightToCheck := networkInfo.Height - self.Config.Checker.MinConfirmationBlocks

		// Interrupts longer queries
		ctx, cancel := context.WithTimeout(self.Ctx, 5*time.Minute)
		defer cancel()

		// Preallocate the slices
		ids := make([]string, 0, self.Config.Checker.MaxBundlesPerRun)

		err = self.db.WithContext(ctx).
			Transaction(func(tx *gorm.DB) error {
				// Select bundles that will get checked
				err = tx.Raw(`UPDATE data_items
					SET state = 'CHECKING', updated_at = NOW()
					WHERE data_item_id IN (
						SELECT data_item_id 
						FROM data_items 
						WHERE state = 'UPLOADED' AND block_height < ? AND service = ?
						ORDER BY block_height ASC, data_item_id ASC
						LIMIT ?
						FOR UPDATE SKIP LOCKED)
					RETURNING data_item_id`, minHeightToCheck, bundlingService.String(), self.Config.Checker.MaxBundlesPerRun).
					Scan(&ids).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get bundles to check")
					return err
				}
				return nil
			})
		if err != nil {
			self.Log.WithError(err).Error("Failed to get data for checking")
			return
		}

		if len(ids) > 0 {
			self.Log.WithField("len", len(ids)).Debug("Polled interactions for checking")
		}

		for _, id := range ids {
			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &Payload{
				BundlerTxId: id,
				Service:     bundlingService,
				Table:       model.TableDataItem,
			}:
			}
		}

		// Update monitoring
		self.monitor.GetReport().Checker.State.BundlesTakenFromDb.Add(uint64(len(ids)))

		// If we got the maximum number of elements, we need to repeat the query
		if len(ids) == self.Config.Checker.MaxBundlesPerRun {
			repeat = true
		}

		return

	}
}

func (self *Poller) handleDataItemRetrying(bundlingService model.BundlingService) task.RepeatedSubtaskFunc {
	return func() (repeat bool, err error) {
		// Interrupts longer queries
		ctx, cancel := context.WithTimeout(self.Ctx, 5*time.Minute)
		defer cancel()

		// Preallocate the slices
		ids := make([]string, 0, self.Config.Checker.MaxBundlesPerRun)

		err = self.db.WithContext(ctx).
			Transaction(func(tx *gorm.DB) error {
				// Select bundles that will get checked
				err = tx.Raw(`UPDATE data_items
					SET updated_at = NOW()
					WHERE data_item_id IN (
						SELECT data_item_id 
						FROM data_items 
						WHERE state = 'CHECKING' AND service = ? AND updated_at < NOW() - ?::interval 
						ORDER BY block_height ASC, data_item_id ASC
						LIMIT ?
						FOR UPDATE SKIP LOCKED)
					RETURNING data_item_id`,
					bundlingService.String(),
					fmt.Sprintf("%d seconds", int((self.Config.Checker.PollerRetryCheckAfter.Seconds()))),
					self.Config.Checker.MaxBundlesPerRun).
					Scan(&ids).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get bundles to check")
					return err
				}
				return nil
			})
		if err != nil {
			self.Log.WithError(err).Error("Failed to get data for checking")
			return
		}

		if len(ids) > 0 {
			self.Log.WithField("len", len(ids)).Debug("Polled interactions for checking")
		}

		for _, id := range ids {
			select {
			case <-self.Ctx.Done():
				return
			case self.Output <- &Payload{
				BundlerTxId: id,
				Service:     bundlingService,
				Table:       model.TableDataItem,
			}:
			}
		}

		// Update monitoring
		self.monitor.GetReport().Checker.State.BundlesTakenFromDb.Add(uint64(len(ids)))

		// If we got the maximum number of elements, we need to repeat the query
		if len(ids) == self.Config.Checker.MaxBundlesPerRun {
			repeat = true
		}

		return

	}
}
