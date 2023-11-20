package model

import (
	"time"

	"github.com/jackc/pgtype"
)

const (
	TableDataItem = "data_items"
)

type DataItem struct {
	// Id of the data item
	DataItemID string `gorm:"primaryKey"`

	// Data item, ready to be sent. This is a nested bundle or a plain data item
	DataItem pgtype.Bytea

	// State of bundle
	State BundleState

	// Which bundling service was used to send this data item
	Service pgtype.Text

	// Block height upon which interaction was bundled. Used to trigger verification later
	BlockHeight pgtype.Int8

	// Response from bundlr.network
	Response pgtype.JSONB

	// Time of the last update to this row
	UpdatedAt time.Time

	// Time of creation
	CreatedAt time.Time
}

func (DataItem) TableName() string {
	return TableDataItem
}
