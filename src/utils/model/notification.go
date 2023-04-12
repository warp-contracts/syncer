package model

import (
	"encoding/json"
	"syncer/src/utils/arweave"

	"github.com/jackc/pgtype"
)

type ContractNotification struct {
	ContractTxId string        `json:"contract_tx_id"`
	Test         bool          `json:"test"`
	Source       string        `json:"source"`
	InitialState pgtype.JSONB  `json:"initial_state"`
	Tags         []arweave.Tag `json:"tags"`
}

func (self *ContractNotification) MarshalBinary() (data []byte, err error) {
	return json.Marshal(self)
}

type InteractionNotification struct {
	ContractTxId string `json:"contract_tx_id"`
	Test         bool   `json:"test"`
	Source       string `json:"source"`
	Interaction  string `json:"interaction"`
}

func (self *InteractionNotification) MarshalBinary() (data []byte, err error) {
	return json.Marshal(self)
}

type AppSyncContractNotification struct {
	ContractTxId   string `json:"contractTxId"`
	Creator        string `json:"creator"`
	Type           string `json:"type"`
	BlockHeight    uint64 `json:"blockHeight"`
	BlockTimestamp uint64 `json:"blockTimestamp"`
	Source         string `json:"source"`
	SyncTimestamp  int64  `json:"syncTimestamp"`
}

func (self *AppSyncContractNotification) MarshalJSON() (data []byte, err error) {
	return json.Marshal(self)
}
