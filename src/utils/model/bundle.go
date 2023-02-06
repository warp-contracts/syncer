package model

import (
	"database/sql"
	"time"
)

// CREATE TABLE "bundle_items" ("interaction_id" bigserial NOT NULL,"state"  bundle_state NOT NULL,"block_height" bigint,"updated_at" timestamptz,PRIMARY KEY ("interaction_id"),CONSTRAINT "fk_bundle_items_interaction" FOREIGN KEY ("interaction_id") REFERENCES "interactions"("id"))
// CREATE INDEX IF NOT EXISTS "idx_bundle_items_block_height" ON "bundle_items" USING btree("block_height" desc) WHERE state != 'ON_ARWEAVE'
type BundleItem struct {
	InteractionID int           `gorm:"primaryKey; not null; comment:Numerical id of the interaction"`
	Interaction   Interaction   // Can be preloaded by gorm, but isn't by default.
	State         BundleState   `gorm:"not null; type: bundle_state; comment:State of bundle"`
	BlockHeight   sql.NullInt64 `gorm:"index:, sort:desc, type:btree, where:state != 'ON_ARWEAVE'; comment:Block height upon which interaction was bundled. Used to trigger verification later."`
	UpdatedAt     time.Time     `gorm:"comment:Time of the last update to this row"`
}

func (BundleItem) TableName() string {
	return "bundle_items"
}
