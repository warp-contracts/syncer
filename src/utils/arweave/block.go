package arweave

import (
	"bytes"
	"fmt"
)

const (
	HEIGHT_2_5 = int64(812970)
)

type Block struct {
	Nonce                    Base64String   `json:"nonce"`
	PreviousBlock            Base64String   `json:"previous_block"`
	Timestamp                int64          `json:"timestamp"`
	LastRetarget             int64          `json:"last_retarget"`
	Height                   int64          `json:"height"`
	Diff                     BigInt         `json:"diff"`
	Hash                     Base64String   `json:"hash"`
	IndepHash                Base64String   `json:"indep_hash"`
	Txs                      []Base64String `json:"txs"`
	TxRoot                   Base64String   `json:"tx_root"`
	TxTree                   interface{}    `json:"tx_tree"`
	HashList                 interface{}    `json:"hash_list"`
	HashListMerkle           Base64String   `json:"hash_list_merkle"`
	WalletList               Base64String   `json:"wallet_list"`
	RewardAddr               RewardAddr     `json:"reward_addr"`
	Tags                     []interface{}  `json:"tags"`
	RewardPool               BigInt         `json:"reward_pool"`
	WeaveSize                string         `json:"weave_size"`
	BlockSize                string         `json:"block_size"`
	CumulativeDiff           BigInt         `json:"cumulative_diff"`
	SizeTaggedTxs            interface{}    `json:"size_tagged_txs"`
	Poa                      POA            `json:"poa"`
	UsdToArRate              []string       `json:"usd_to_ar_rate"`
	ScheduledUsdToArRate     []string       `json:"scheduled_usd_to_ar_rate"`
	Packing25Threshold       string         `json:"packing_2_5_threshold"`
	StrictDataSplitThreshold string         `json:"strict_data_split_threshold"`
}

type POA struct {
	Option   string       `json:"option"`
	TxPath   Base64String `json:"tx_path"`
	DataPath Base64String `json:"data_path"`
	Chunk    Base64String `json:"chunk"`
}

func (b *Block) IsValid() bool {
	if b.Height < HEIGHT_2_5 {
		return false
	}

	bds := generateBlockDataSegment(b)
	list := []any{
		bds,
		b.Hash,
		b.Nonce,
		[]interface{}{
			b.Poa.Option,
			b.Poa.TxPath,
			b.Poa.DataPath,
			b.Poa.Chunk,
		},
	}
	hash := DeepHash(list)

	return bytes.Equal(hash[:], b.IndepHash)
}

func generateBlockDataSegment(b *Block) []byte {
	bdsBase := generateBlockDataSegmentBase(b)

	values := []any{
		bdsBase,
		fmt.Sprintf("%d", b.Timestamp),
		fmt.Sprintf("%d", b.LastRetarget),
		b.Diff.String(),
		b.CumulativeDiff.String(),
		b.RewardPool.String(),
		b.WalletList,
		b.HashListMerkle,
	}
	hash := DeepHash(values)
	return hash[:]
}

func generateBlockDataSegmentBase(b *Block) []byte {
	props := []any{
		b.UsdToArRate[0],          //RateDividend
		b.UsdToArRate[1],          //RateDivisor
		b.ScheduledUsdToArRate[0], //ScheduledRateDividend
		b.ScheduledUsdToArRate[1], //ScheduledRateDivisor
		b.Packing25Threshold,
		b.StrictDataSplitThreshold,
		fmt.Sprintf("%d", b.Height),
		b.PreviousBlock,
		b.TxRoot,
		b.Txs,
		b.BlockSize,
		b.WeaveSize,
		b.RewardAddr,
		b.Tags,
	}

	hash := DeepHash(props)
	return hash[:]
}
