package model

type State struct {
	Id                         int   `gorm:"not null; check:ensure_one_row,id=1; comment:Id always equals zero"`
	LastTransactionBlockHeight int64 `gorm:"not null; default:1007373; comment:Height of the last fully processed transaction block"`
}

func (State) TableName() string {
	return "syncer_state"
}
