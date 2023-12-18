package warp

import (
	"bytes"
	"errors"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/smartweave"

	"golang.org/x/exp/slices"
)

var supportedContentType = []string{"application/javascript", "application/wasm"}

func SetContractSourceMetadata(sourceTx *arweave.Transaction, out *model.ContractSource) (err error) {
	out.SrcTxId = sourceTx.ID.Base64()

	err = out.DeploymentType.Set("arweave")
	if err != nil {
		return
	}

	srcContentType, ok := sourceTx.GetTag(smartweave.TagContentType)
	if !ok {
		err = errors.New("contract source content type is not set")
		return
	}

	if !slices.Contains(supportedContentType, srcContentType) {
		err = errors.New("unsupported contract source content type")
		return
	}

	err = out.SrcContentType.Set(srcContentType)
	if err != nil {
		return
	}

	err = out.SrcTx.Set(sourceTx)
	if err != nil {
		return
	}

	// Check signature
	err = sourceTx.Verify()
	if err != nil {
		return
	}

	// Set owner
	owner, err := GetWalletAddress(sourceTx)
	if err != nil {
		return
	}

	err = out.Owner.Set(owner)
	if err != nil {
		return
	}

	return
}

func SetContractSource(src bytes.Buffer, srcTx *arweave.Transaction, out *model.ContractSource) (err error) {
	if out.IsJS() {
		err = out.Src.Set(src.String())
		if err != nil {
			return
		}
	} else {
		srcWasmLang, ok := srcTx.GetTag(TagWasmLang)
		if !ok {
			err = errors.New("WASM contract language is not set")
			return
		}
		err = out.SrcWasmLang.Set(srcWasmLang)
		if err != nil {
			return
		}

		err = out.SrcBinary.Set(src.Bytes())
		if err != nil {
			return
		}
	}

	return
}
