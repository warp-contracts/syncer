package sync

import "syncer/src/utils/model"

type Payload struct {
	BlockHeight  int64
	Interactions []*model.Interaction
}
