package interact

import (
	"time"

	"github.com/warp-contracts/syncer/src/utils/bundlr"
)

type Payload struct {
	DataItem  *bundlr.BundleItem
	Timestamp time.Time
}
