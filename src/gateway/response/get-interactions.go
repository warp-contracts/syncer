package response

import (
	"encoding/json"

	"github.com/warp-contracts/syncer/src/utils/model"
)

type Interaction struct {
	ContractId  string          `json:"contractTxId"`
	SortKey     string          `json:"sortKey"`
	LastSortKey string          `json:"lastSortKey"`
	Interaction json.RawMessage `json:"interaction"`
}

type GetInteractions struct {
	Interactions []Interaction `json:"interactions"`
}

// Returns unchanged interaction upon error
func addSortKeyToInteraction(interactionBytes []byte, sortKey string) (out []byte) {
	var objmap map[string]interface{}

	err := json.Unmarshal(interactionBytes, &objmap)
	if err != nil {
		return interactionBytes
	}

	objmap["sortKey"] = sortKey

	out, err = json.Marshal(objmap)
	if err != nil {
		return interactionBytes
	}

	return
}

func InteractionsToResponse(interactions []*model.Interaction) *GetInteractions {
	out := make([]Interaction, len(interactions))
	for i, interaction := range interactions {
		// Add sort key to Interaction json
		out[i] = Interaction{
			ContractId:  interaction.ContractId,
			SortKey:     interaction.SortKey,
			LastSortKey: interaction.LastSortKey.String,
			Interaction: addSortKeyToInteraction(interaction.Interaction.Bytes, interaction.SortKey),
		}
	}

	return &GetInteractions{
		Interactions: out,
	}
}
