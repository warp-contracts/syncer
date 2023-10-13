package gateway

import (
	"database/sql"
	"errors"
	"net/http"
	"time"

	"github.com/warp-contracts/syncer/src/utils/binder"
	. "github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/utils/model"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/warp-contracts/syncer/src/gateway/request"
	"github.com/warp-contracts/syncer/src/gateway/response"
)

var ErrWindowOverlapsLastBlock = errors.New("window overlaps last block")

func (self *Server) onGetInteractions(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in = new(request.GetInteractions)
		err := c.ShouldBindWith(in, binder.JSON)
		if err != nil {
			LOGE(c, err, http.StatusBadRequest).Error("Failed to parse request")
			return
		}

		// Defaults
		if in.Limit == 0 {
			in.Limit = 10000
		}

		// Wait for the current widnow to finish if End is in the future
		delta := int64(in.End) - int64(time.Now().UnixMilli())
		delta += 2000 // 2s margin for clock skew between GW and DB
		delta += 100  // 100ms margin of the request handling done so far
		if delta > 0 {
			if delta > self.Config.Gateway.ServerRequestTimeout.Milliseconds() {
				// Request would timeout before the window is finished
				LOGE(c, nil, http.StatusBadRequest).Error("End timestamp is too far in the future")
				return
			}

			t := time.NewTimer(time.Duration(delta) * time.Millisecond)
			LOG(c).WithField("duration", delta).Trace("Waiting for the window to finish")
			select {
			case <-c.Done():
				// Request is cancelled
				return
			case <-t.C:
				// Window is finished, continue
			}
		}

		LOG(c).WithField("start", in.Start).WithField("end", in.End).Debug("Get interactions")

		var interactions []*model.Interaction

		err = db.WithContext(c).
			Transaction(func(tx *gorm.DB) (err error) {
				// Get Forwarder state, this is the last block height that has last_sort_key set
				var forwarderState model.State
				err = tx.WithContext(self.Ctx).
					Where("name = ?", model.SyncedComponentForwarder).
					First(&forwarderState).
					Error
				if err != nil {
					self.Log.WithError(err).Error("Failed to get forwarder state")
					return err
				}

				// Check if the time window is finished and it isn't contained
				var isOverlapping int
				tx.Raw(`SELECT 1 FROM interactions
				WHERE interactions.sync_timestamp >= ?
				AND interactions.sync_timestamp < ?
				AND interactions.block_height > ?				
				LIMIT 1`, in.Start, in.End, forwarderState.FinishedBlockHeight).
					Scan(&isOverlapping)
				if isOverlapping > 0 {
					return ErrWindowOverlapsLastBlock
				}

				// Get interactions
				query := tx.WithContext(c).
					Table(model.TableInteraction).
					Where("interactions.sync_timestamp >= ?", in.Start).
					Where("interactions.sync_timestamp < ?", in.End).
					Where("interactions.block_height <= ?", forwarderState.FinishedBlockHeight).
					Limit(in.Limit).
					Offset(in.Offset).
					Order("interactions.sort_key ASC")

				if len(in.BlacklistedContracts) > 0 {
					query = query.Where("interactions.contract_id NOT IN ?", in.BlacklistedContracts)
				}

				if len(in.SrcIds) > 0 {
					query = query.Where("contracts.src_tx_id IN ?", in.SrcIds).
						Joins("JOIN contracts ON interactions.contract_id = contracts.contract_id")
				}

				err = query.Find(&interactions).Error
				return
			}, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
		if err != nil {
			if errors.Is(err, ErrWindowOverlapsLastBlock) {
				c.Status(http.StatusNoContent)
				LOG(c).Debug("Window overlaps with the current block, returning 204")
				return
			}

			LOGE(c, err, http.StatusInternalServerError).Error("Failed to fetch interactions")
			// Update monitoring
			self.monitor.GetReport().Gateway.Errors.DbError.Inc()
			return
		}
		LOG(c).WithField("start", in.Start).
			WithField("end", in.End).
			WithField("num", len(interactions)).
			Debug("Return interactions")

		// Update monitoring
		self.monitor.GetReport().Gateway.State.InteractionsReturned.Add(uint64(len(interactions)))

		c.JSON(http.StatusOK, response.InteractionsToResponse(interactions))
	}
}
