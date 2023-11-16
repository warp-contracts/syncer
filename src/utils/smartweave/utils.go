package smartweave

import (
	"errors"
	"fmt"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/tool"
)

func ValidateInteraction(tx *arweave.Transaction) error {
	if tx.Format < 2 {
		return errors.New("interaction should not have a format smaller than 2")
	}

	tags := tx.GetTagMap(TagAppName, TagContractTxId, TagInput)

	if string(tags[TagAppName]) != TagAppNameValue {
		return fmt.Errorf("interaction should have a tag '%s' with a value '%s'", TagAppName, TagAppNameValue)
	}

	if !TagContractTxIdRegex.Match(tags[TagContractTxId]) {
		return errors.New("interaction contract id is not in the correct format")
	}

	if tool.CheckJSON(tags[TagInput]) != nil {
		return fmt.Errorf("value of the tag '%s' is not a valid JSON", TagInput)
	}

	return nil
}
