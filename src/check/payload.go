package check

import "github.com/warp-contracts/syncer/src/utils/model"

type Payload struct {
	InteractionId int
	BundlerTxId   string
	Service       model.BundlingService
	Table         string
}
