package warpy_sync

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/core/types"
)

type Input struct {
	Function              string   `json:"function"`
	Points                int64    `json:"points"`
	AdminId               string   `json:"adminId"`
	Members               []Member `json:"members"`
	NoBoost               bool     `json:"noBoost"`
	Cap                   int64    `json:"cap"`
	InitialCapInteraction bool     `json:"initialCapInteraction"`
}

func (input Input) MarshalJSON() ([]byte, error) {
	type WarpySyncInputAlias Input
	return json.Marshal(&struct {
		*WarpySyncInputAlias
	}{WarpySyncInputAlias: (*WarpySyncInputAlias)(&input)})
}

type Member struct {
	Id     string   `json:"id"`
	Roles  []string `json:"roles"`
	TxId   string   `json:"txId"`
	Points int64    `json:"points"`
}

type BlockInfoPayload struct {
	Transactions []*types.Transaction
	Height       uint64
	Hash         string
	Timestamp    uint64
}

type LastSyncedBlockPayload struct {
	Height    uint64
	Hash      string
	Timestamp uint64
}

type InteractionPayload struct {
	FromAddress string
	Points      int64
}

type SommelierTransactionPayload struct {
	Transaction *types.Transaction
	FromAddress string
	Block       *BlockInfoPayload
	Method      *abi.Method
	ParsedInput []byte
	Input       map[string]interface{}
	Assets      float64
}
