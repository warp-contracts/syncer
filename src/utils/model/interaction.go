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
	SortKey            string

	// https://github.com/warp-contracts/gateway/blob/main/src/gateway/tasks/syncTransactions.ts#L175
	Evolve sql.NullString

	// https://github.com/warp-contracts/gateway/blob/ef7aad549045943f0127542cce36cd94a966bdc7/src/gateway/tasks/syncTransactions.ts#L187
	Testnet sql.NullString

	// Hardcoded arsyncer
	Source string

	// Wallet address, 44 characters
	Owner string

	State InteractionState

	// Those fields aren't used anymore, currenct version waits 10 blocks with the synchronization
	// ConfirmingPeer    string
	// ConfirmedAtHeight int64
	// Confirmations     string

	// Not needed:
	// BundlerTxId       string
	// LastSortKey       string
}
