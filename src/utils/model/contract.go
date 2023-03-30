package model

import (
	"github.com/jackc/pgtype"
)

const (
	TableContract = "contracts"

	ContractTypePst   = "pst"
	ContractTypeOther = "other"
)

type Contract struct {
	ContractId          string
	SrcTxId             pgtype.Varchar
	InitState           pgtype.JSONB
	Owner               pgtype.Varchar
	Type                pgtype.Varchar
	Project             pgtype.Varchar
	PstTicker           pgtype.Varchar
	PstName             pgtype.Varchar
	BlockHeight         uint64 // NOT NULL
	ContentType         pgtype.Varchar
	BundlerContractTxId pgtype.Varchar
	BundlerContractNode pgtype.Varchar
	ContractTx          pgtype.JSONB
	BundlerContractTags pgtype.JSONB
	BlockTimestamp      uint64 // NOT NULL
	Testnet             pgtype.Varchar
	BundlerResponse     pgtype.Varchar
	DeploymentType      pgtype.Varchar
	Manifest            pgtype.JSONB
}
