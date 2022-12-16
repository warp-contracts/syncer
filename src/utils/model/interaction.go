package model

import (
	"context"

	"gorm.io/gorm"
)

const (
	TableInteraction = "interactions"
)

type Interaction struct {
	InteractionId      string
	Interaction        string
	BlockHeight        int64
	BlockId            string
	ContractId         string
	Function           string
	Input              string
	ConfirmationStatus string
}

func LastBlockHeight(ctx context.Context, db *gorm.DB) (out int64, err error) {
	//FIXME: This is a wrong query, it should account for transactions that aren't interactions
	err = db.WithContext(ctx).
		Table(TableInteraction).
		Select("MAX(block_height)").
		Row().
		Scan(&out)
	return
}
