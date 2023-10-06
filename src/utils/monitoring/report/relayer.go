package report

import (
	"go.uber.org/atomic"
)

type RelayerErrors struct {
}

type RelayerState struct {
	BlocksReceived atomic.Uint64 `json:"blocks_received"`

	BundleItemsSaved atomic.Uint64 `json:"bundle_items_saved"`
}

type RelayerReport struct {
	State  RelayerState  `json:"state"`
	Errors RelayerErrors `json:"errors"`
}
