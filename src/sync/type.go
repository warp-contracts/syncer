package sync

import (
	"syncer/src/utils/arweave"
	"syncer/src/utils/model"
)

type Payload struct {
	BlockHeight  uint64
	BlockHash    arweave.Base64String
	Interactions []*model.Interaction
}
