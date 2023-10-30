package warp

import (
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
)

func IsL1Interaction(tx *arweave.Transaction) bool {
	if tx.Format < 2 {
		return false
	}

	for _, tag := range tx.Tags {
		if string(tag.Value) == "SmartWeaveAction" &&
			string(tag.Name) == smartweave.TagAppName {
			return true
		}
	}

	return false
}
