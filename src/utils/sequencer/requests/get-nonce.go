package requests

import "github.com/warp-contracts/syncer/src/utils/bundlr"

type GetNonce struct {
	SignatureType bundlr.SignatureType `json:"signature_type"`
	Owner         string               `json:"owner"`
}
