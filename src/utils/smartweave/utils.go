package smartweave

import (
	"errors"
	"fmt"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/tool"
)

func ValidateInteraction(tx *arweave.Transaction) (isInteraction bool, err error) {
	if tx == nil || tx.Format < 2 {
		return false, nil
	}

	hasContractTag := false
	for _, tag := range tx.Tags {
		switch string(tag.Name) {
		case TagAppName:
			if string(tag.Value) == TagAppNameValue {
				isInteraction = true
			}
		case TagInput:
			// Input tag must be a valid JSON
			if tool.CheckJSON(tag.Value) != nil {
				err = fmt.Errorf("value of the tag '%s' is not a valid JSON", TagInput)
				break
			}
		case TagContractTxId:
			hasContractTag = true
			if !TagContractTxIdRegex.Match(tag.Value) {
				err = errors.New("interaction contract id is not in the correct format")
				break
			}
		}
	}

	if !isInteraction {
		return false, nil
	}

	if err != nil {
		return true, err
	}

	if !hasContractTag {
		return true, fmt.Errorf("interaction should have a tag '%s'", TagContractTxId)
	}

	return true, nil
}
