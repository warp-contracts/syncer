package model

import "syncer/src/utils/arweave"

type State struct {
	// Id always equals one
	Id int

	// Height of the last fully processed transaction block
	LastTransactionBlockHeight int64

	// Hash of the last fully processed transaction block
	// Next block needs to have this hash set as its previous block hash
	LastProcessedBlockHash arweave.Base64String

	// Height of the last fully processed transaction block
	ContractLastTransactionBlockHeight uint64

	// Hash of the last fully processed transaction block
	// Next block needs to have this hash set as its previous block hash
	ContractLastProcessedBlockHash arweave.Base64String
}

func (State) TableName() string {
	return "syncer_state"
}
