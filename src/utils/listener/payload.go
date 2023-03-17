package listener

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/model"
)

type Payload struct {
	BlockHash      arweave.Base64String
	BlockHeight    int64
	BlockTimestamp int64
	Interactions   []*model.Interaction
	Transactions   []*arweave.Transaction
}
