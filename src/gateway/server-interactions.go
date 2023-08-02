package gateway

import (
	"net/http"

	. "github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/utils/model"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
	"github.com/warp-contracts/syncer/src/gateway/request"
	"github.com/warp-contracts/syncer/src/gateway/response"
)

func (self *Server) onGetInteractions(c *gin.Context) {
	var in request.GetInteractions
	err := c.ShouldBindJSON(&in)
	if err != nil {
		LOGE(c, err, http.StatusBadRequest).Error("Failed to parse request")
	}

	// Defaults
	if in.Limit == 0 {
		in.Limit = 10000
	}

	var interactions []*model.Interaction
	err = self.db.WithContext(self.Ctx).
		Transaction(func(tx *gorm.DB) (err error) {
			query := self.db.Table(model.TableInteraction).
				Where("sync_timestamp >= ?", in.Start).
				Where("sync_timestamp < ?", in.End).
				Limit(in.Limit).
				Offset(in.Offset).
				Order("sort_key ASC")

			if len(in.SrcIds) > 0 {
				query = query.Where("src_id IN ?", in.SrcIds)
			}

			err = query.Find(&interactions).Error

			return
		})
	if err != nil {
		LOGE(c, err, http.StatusInternalServerError).Error("Failed to fetch interactions")
		return
	}

	c.JSON(http.StatusOK, response.InteractionsToResponse(interactions))
}
