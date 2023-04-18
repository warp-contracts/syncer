package model

import "syncer/src/utils/arweave"

const TableState = "sync_state"

type State struct {
	// Name of the synced component (interactions, contracts)
	Name SyncedComponent `gorm:"primaryKey"`

	// Height of the last fully processed transaction block
	FinishedBlockHeight uint64

	// Hash of the last fully processed transaction block
	// Next block needs to have this hash set as its previous block hash
	FinishedBlockHash arweave.Base64String
}

func (State) TableName() string {
	return TableState
}
