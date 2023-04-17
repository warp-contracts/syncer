package model

import "syncer/src/utils/arweave"

const TableState = "syncer_state"

type State struct {
	// Id always equals one
	Id int

	// Height of the last fully processed transaction block
	LastTransactionBlockHeight uint64

	// Hash of the last fully processed transaction block
	// Next block needs to have this hash set as its previous block hash
	LastProcessedBlockHash arweave.Base64String

	// Height of the last fully processed transaction block
	ContractFinishedHeight uint64

	// Hash of the last fully processed transaction block
	// Next block needs to have this hash set as its previous block hash
	ContractFinishedBlockHash arweave.Base64String
}

func (State) TableName() string {
	return TableState
}
