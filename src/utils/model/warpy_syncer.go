package model

import (
	"github.com/jackc/pgtype"
)

const TableWarpySyncerTransactions = "warpy_syncer_transactions"
const TableWarpySyncerAssets = "warpy_syncer_assets"

type WarpySyncerTransaction struct {
	TxId           string       `gorm:"primaryKey" json:"tx_id"`
	FromAddress    string       `json:"from_address"`
	ToAddress      string       `json:"to_address"`
	BlockHeight    uint64       `json:"block_height"`
	BlockTimestamp uint64       `json:"block_timestamp"`
	SyncTimestamp  uint64       `json:"sync_timestamp"`
	MethodName     string       `json:"method_name"`
	Chain          string       `json:"chain"`
	Protocol       string       `json:"protocol"`
	Input          pgtype.JSONB `json:"input"`
}

type WarpySyncerAssets struct {
	TxId        string  `gorm:"primaryKey" json:"tx_id"`
	FromAddress string  `json:"from_address"`
	Assets      float64 `json:"assets"`
	Timestamp   uint64  `json:"timestamp"`
	Protocol    string  `json:"protocol"`
	Chain       string  `json:"chain"`
}

type WalletDiscordIdPayload struct {
	WalletToDiscordId map[string]string `json:"wallet_to_id"`
}

type DiscordIdRolesPayload struct {
	IdToRoles map[string][]string `json:"id_to_roles"`
}
