package redstone_tx_sync

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/core/types"
)

type Input struct {
	Function string   `json:"function"`
	Points   int      `json:"points"`
	AdminId  string   `json:"adminId"`
	Members  []Member `json:"members"`
	NoBoost  bool     `json:"noBoost"`
}

func (input Input) MarshalJSON() ([]byte, error) {
	type RedstoneTxSyncInputAlias Input
	return json.Marshal(&struct {
		*RedstoneTxSyncInputAlias
	}{RedstoneTxSyncInputAlias: (*RedstoneTxSyncInputAlias)(&input)})
}

type Member struct {
	Id    string   `json:"id"`
	Roles []string `json:"roles"`
}

type Payload struct {
	Transactions types.Transactions
	BlockHeight  int64
	BlockHash    string
}
