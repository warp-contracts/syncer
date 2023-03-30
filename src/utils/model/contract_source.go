package model

import "github.com/jackc/pgtype"

type ContractSource struct {
	SrcTxId         string
	Src             string
	SrcContentType  string
	SrcBinary       pgtype.Bytea
	SrcWasmLang     string
	BundlerSrcTxId  string
	BundlerSrcNode  string
	SrcTx           pgtype.JSONB
	Owner           string
	Testnet         string
	BundlerResponse string
	DeploymentType  string
}

func (ContractSource) TableName() string {
	return "contract_src"
}

func (self *ContractSource) IsJS() bool {
	return self.SrcContentType == "application/javascript"
}
