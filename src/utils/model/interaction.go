package model

import (
	"database/sql"

	"github.com/lib/pq"
)

const (
	TableInteraction = "interactions"
)

type Interaction struct {
	InteractionId      string
	Interaction        string
	BlockHeight        int64
	BlockId            string
	ContractId         string
	Function           string
	Input              string
	ConfirmationStatus string
	InteractWrite      pq.StringArray `gorm:"type:text[]"`

	// https://github.com/warp-contracts/gateway/blob/main/src/gateway/tasks/syncTransactions.ts#L175
	Evolve sql.NullString

	// https://github.com/warp-contracts/gateway/blob/ef7aad549045943f0127542cce36cd94a966bdc7/src/gateway/tasks/syncTransactions.ts#L187
	Testnet sql.NullString

	// Hardcoded arsyncer
	Source string

	// TODO: Generate this, gateway and sequencer should use the same function
	// https://github.com/warp-contracts/sequencer/blob/test2/sortkey/createSortKey.go
	// L1 sdk has a slighly different code. Do it exatctly like SDK
	// Środkowa część same zera ( w SDK jest https://github.com/warp-contracts/warp/blob/main/src/core/modules/impl/LexicographicalInteractionsSorter.ts#L35)
	// TODO: Exact height of the block isn't yet known
	SortKey string

	// TODO: This should be wallet address (there's a function in goar to do this), 44char
	Owner string

	// Those fields aren't used anymore, currenct version waits 10 blocks with the synchronization
	// ConfirmingPeer    string
	// ConfirmedAtHeight int64
	// Confirmations     string

	// Not needed:
	// BundlerTxId       string
	// LastSortKey       string

}
