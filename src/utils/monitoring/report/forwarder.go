package report

import (
	"go.uber.org/atomic"
)

type ForwarderErrors struct {
}

type ForwarderState struct {
	CurrentHeight           atomic.Uint64 `json:"current_height"`
	FinishedHeight          atomic.Uint64 `json:"finished_height"`
	L1InteractionsPublished atomic.Uint64 `json:"l1_interactions_published"`
	L2InteractionsPublished atomic.Uint64 `json:"l2_interactions_published"`
	L1BroadcastSeconds      atomic.Uint64 `json:"l1_broadcast_seconds"`
	L2BroadcastSeconds      atomic.Uint64 `json:"l2_broadcast_seconds"`
}

type ForwarderReport struct {
	State  ForwarderState  `json:"state"`
	Errors ForwarderErrors `json:"errors"`
}
