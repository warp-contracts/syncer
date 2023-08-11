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

func InteractionsToResponse(interactions []*model.Interaction) *GetInteractions {
	out := make([]Interaction, len(interactions))
	for i, interaction := range interactions {
		out[i] = Interaction{
			ContractId:  interaction.ContractId,
			SortKey:     interaction.SortKey,
			LastSortKey: interaction.LastSortKey.String,
			Interaction: interaction.Interaction.Bytes,
		}
	}

	return &GetInteractions{
		Interactions: out,
	}
}
