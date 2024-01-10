package report

import (
	"go.uber.org/atomic"
)

type CheckerErrors struct {
	IrysGetStatusError  atomic.Uint64 `json:"irys_check_state_error"`
	TurboGetStatusError atomic.Uint64 `json:"turbo_get_status_error"`
	DbStateUpdateError  atomic.Uint64 `json:"db_state_update_error"`
}

type CheckerState struct {
	BundlesTakenFromDb     atomic.Uint64 `json:"bundles_taken_from_db"`
	AllCheckedBundles      atomic.Uint64 `json:"all_checked_bundles"`
	FinishedBundles        atomic.Uint64 `json:"finished_bundles"`
	UnfinishedBundles      atomic.Uint64 `json:"unfinished_bundles"`
	IrysUnfinishedBundles  atomic.Uint64 `json:"irys_unfinished_bundles"`
	TurboUnfinishedBundles atomic.Uint64 `json:"turbo_unfinished_bundles"`
	DbStateUpdated         atomic.Uint64 `json:"db_state_updated"`
}

type CheckerReport struct {
	State  CheckerState  `json:"state"`
	Errors CheckerErrors `json:"errors"`
}
