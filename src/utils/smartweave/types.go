package smartweave

import "syncer/src/utils/arweave"

type Owner struct {
	Address string `json:"address"`
}

type Block struct {
	Height    int64  `json:"height"`
	Id        string `json:"id"`
	Timestamp int64  `json:"timestamp"`
}

type Amount struct {
	Winston string `json:"winston"`
}

type Interaction struct {
	Id        string               `json:"id"`
	Owner     Owner                `json:"owner"`
	Recipient arweave.Base64String `json:"recipient"`
	Tags      []arweave.Tag        `json:"tags"`
	Block     Block                `json:"block"`
	Fee       Amount               `json:"fee"`
	Quantity  Amount               `json:"quantity"`
}
