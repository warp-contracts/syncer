package contract

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/model"
)

type ContractData struct {
	Contract *model.Contract
	Source   *model.ContractSource
}

type Payload struct {
	BlockHeight uint64
	BlockHash   arweave.Base64String
	Data        []*ContractData
}
