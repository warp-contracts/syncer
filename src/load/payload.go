package load

import "syncer/src/utils/model"

type Payload struct {
	Interaction *model.Interaction
	BundleItem  *model.BundleItem
}
