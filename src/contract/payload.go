package contract

import (
	"syncer/src/utils/model"
)

type Payload struct {
	Contract *model.Contract
	Source   *model.ContractSource
}
