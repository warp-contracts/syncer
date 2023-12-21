package smartweave

import "github.com/warp-contracts/syncer/src/utils/arweave"

type Owner struct {
	Address string `json:"address"`
}

type Block struct {
	Height    int64                `json:"height"`
	Id        arweave.Base64String `json:"id"`
	Timestamp int64                `json:"timestamp"`
}

type SequencerBlock struct {
	Height    int64 `json:"height"`
	Timestamp int64 `json:"timestamp"`
}

type Amount struct {
	Winston string `json:"winston"`
}

type Tag struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Interaction struct {
	Id             arweave.Base64String `json:"id"`
	Owner          Owner                `json:"owner"`
	Recipient      arweave.Base64String `json:"recipient"`
	Tags           []Tag                `json:"tags"`
	Block          Block                `json:"block"`
	Fee            Amount               `json:"fee"`
	Quantity       Amount               `json:"quantity"`
	Random         arweave.Base64String `json:"random,omitempty"`
	SequencerBlock *SequencerBlock      `json:"sequencer_block,omitempty"`
}
