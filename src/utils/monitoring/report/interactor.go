package report

import (
	"go.uber.org/atomic"
)

type InteractorErrors struct {
	SenderError    atomic.Int64 `json:"sender_error"`
	CheckerDbError atomic.Int64 `json:"checker_db_error"`
}

type InteractorState struct {
	CheckerDelay        atomic.Int64 `json:"checker_delay"`
	SenderSentDataItems atomic.Int64 `json:"sender_sent_data_items"`
}

type InteractorReport struct {
	State  InteractorState  `json:"state"`
	Errors InteractorErrors `json:"errors"`
}
