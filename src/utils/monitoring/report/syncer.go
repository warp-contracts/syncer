package report

import (
	"go.uber.org/atomic"
)

type SyncerErrors struct {
	DbInteractionInsert               atomic.Int64 `json:"db_interaction"`
	DbLastTransactionBlockHeightError atomic.Int64 `json:"db_last_tx_block_height"`
	TxValidationErrors                atomic.Int64 `json:"tx_validation"`
}

type SyncerState struct {
	SyncerFinishedHeight atomic.Int64 `json:"syncer_finished_height"`

	AverageInteractionsSavedPerMinute atomic.Float64 `json:"average_interactions_saved_per_minute"`

	InteractionsSaved        atomic.Uint64 `json:"interactions_saved"`
	FailedInteractionParsing atomic.Uint64 `json:"failed_interaction_parsing"`
}

type SyncerReport struct {
	State  SyncerState  `json:"state"`
	Errors SyncerErrors `json:"errors"`
}
