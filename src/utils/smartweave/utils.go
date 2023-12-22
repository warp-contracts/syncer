package smartweave

import (
	"errors"
	"fmt"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/tool"
)

const MaxInteractionDataItemSizeBytes = 20000

func ValidateInteraction(tx *arweave.Transaction, isInputInDataEnabled bool) (isInteraction bool, err error) {
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
			switch string(tag.Value) {
			case TagInputFormatTagValue:
				inputFromTag = true
			case TagInputFormatDataValue:
				inputFromTag = false
			default:
				err = fmt.Errorf("%s' tag value can only be '%s' or '%s'",
					TagInputFormat, TagInputFormatTagValue, TagInputFormatDataValue)
			}
		case TagContractTxId:
			hasContractTag = true
			if !arweave.TxIdRegex.Match(tag.Value) {
				err = errors.New("interaction contract id is not in the correct format")
			}
		}
	}

	if !isInteraction {
		return false, nil
	}

	if !inputFromTag {
		if !isInputInDataEnabled {
			// Input in data is disabled
			return false, nil
		}
		input = tx.Data
	}
	// Input must be a valid JSON
	if jsonError := tool.CheckJSON(input); jsonError != nil {
		err = fmt.Errorf("value of the input is not a valid JSON: %s", jsonError.Error())
	}

	if tx.Size() > MaxInteractionDataItemSizeBytes {
		err = fmt.Errorf("the size of the interaction exceeds the limit: %d bytes", MaxInteractionDataItemSizeBytes)
	}

	if err != nil {
		return true, err
	}

	if !hasContractTag {
		return true, fmt.Errorf("interaction should have a tag '%s'", TagContractTxId)
	}

	return true, nil
}

// whether the interaction should have a non-empty data field
func IsInteractionWithData(tx *arweave.Transaction) bool {
	if tx == nil || tx.Format < 2 {
		return false
	}

	dataSize := tx.DataSize.Int64()
	if dataSize <= 0 || dataSize > MaxInteractionDataItemSizeBytes {
		return false
	}

	inputFormat, ok := tx.GetTag(TagInputFormat)
	return ok && inputFormat == TagInputFormatDataValue
}
