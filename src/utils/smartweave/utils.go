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
	var input arweave.Base64String
	inputFromTag := true

	for _, tag := range tx.Tags {
		switch string(tag.Name) {
		case TagAppName:
			if string(tag.Value) == TagAppNameValue {
				isInteraction = true
			}
		case TagInput:
			input = tag.Value
		case TagInputFormat:
			if string(tag.Value) == TagInputFormatTagValue {
				inputFromTag = true
			} else if string(tag.Value) == TagInputFormatDataValue {
				inputFromTag = false
			} else {
				err = fmt.Errorf("%s' tag value can only be '%s' or '%s'",
					TagInputFormat, TagInputFormatTagValue, TagInputFormatDataValue)
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

	if !inputFromTag {
		input = tx.Data
	}
	// Input must be a valid JSON
	if jsonError := tool.CheckJSON(input); jsonError != nil {
		err = fmt.Errorf("value of the input is not a valid JSON: %s", jsonError.Error())
	}

	if err != nil {
		return true, err
	}

	if !hasContractTag {
		return true, fmt.Errorf("interaction should have a tag '%s'", TagContractTxId)
	}

	return true, nil
}
