package forward

import "github.com/jackc/pgtype"

type Notification struct {
	Interaction pgtype.JSONB `json:"interaction"`
	ContractId  string       `json:"contractId"`
	SrcTxId     string       `json:"srcTxId"`
}
