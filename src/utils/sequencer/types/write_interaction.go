package types

import (
	"encoding/json"

	"github.com/warp-contracts/syncer/src/utils/bundlr"
)

type WriteInteractionOptions struct {
	Vrf           bool                 `json:"vrf"`
	Tags          bundlr.Tags          `json:"tags"`
	ContractTxId  string               `json:"contractTxId"`
	ManifestData  ManifestDataType     `json:"manifestData"`
	SignatureType bundlr.SignatureType `json:"signatureType"`
}

type InteractionDataField struct {
	Input    json.Marshaler   `json:"input"`
	Manifest ManifestDataType `json:"manifest"`
}

type ManifestDataType map[string]string
