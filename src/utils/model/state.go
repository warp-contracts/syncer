package model

type State struct {
	// Id always equals one
	Id int

	// Height of the last fully processed transaction block
	LastTransactionBlockHeight int64
}

func (State) TableName() string {
	return "syncer_state"
}
