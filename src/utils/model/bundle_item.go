package model

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"syncer/src/utils/arweave"
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

func (self *BundleItem) GetDataItem() (out []byte, err error) {
	owner, err := base64.RawURLEncoding.DecodeString(self.Interaction.Owner)
	if err != nil {
		return
	}

	tx := arweave.Transaction{
		ID:    self.Interaction.InteractionId,
		Owner: owner,
	}

	// Format    int          `json:"format"`
	// ID        string       `json:"id"`
	// LastTx    Base64String `json:"last_tx"`
	// Owner     Base64String `json:"owner"` // utils.Base64Encode(wallet.PubKey.N.Bytes())
	// Tags      []Tag        `json:"tags"`
	// Target    Base64String `json:"target"`
	// Quantity  string       `json:"quantity"`
	// Data      string       `json:"data"` // base64.encode
	// DataSize  string       `json:"data_size"`
	// DataRoot  Base64String `json:"data_root"`
	// Reward    string       `json:"reward"`
	// Signature Base64String `json:"signature"`

	return json.Marshal(tx)
}
