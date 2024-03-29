package relay

import (
	"encoding/base64"
	"strconv"

	"github.com/jackc/pgtype"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/model"
)

func getTags(payload *Payload, source, env string, interaction *model.Interaction, random []byte) (out []bundlr.Tag) {
	out = []bundlr.Tag{
		// Global
		{Name: "Sequencer", Value: "Warp"},
		{Name: "Source", Value: source},
		{Name: "Env", Value: env},
		// Block specific
		{Name: "Arweave-Block-Height", Value: strconv.FormatUint(payload.LastArweaveBlock.Height, 10)},
		{Name: "Arweave-Block-Timestamp", Value: strconv.FormatUint(payload.LastArweaveBlock.Timestamp, 10)},
		{Name: "Arweave-Block-Hash", Value: payload.LastArweaveBlock.Hash},
		{Name: "Sequencer-Height", Value: strconv.FormatInt(payload.SequencerBlockHeight, 10)},
		{Name: "Sequencer-Timestamp", Value: strconv.FormatInt(payload.SequencerBlockTimestamp, 10)},
		// Interaction specific
		{Name: "Sort-Key", Value: interaction.SortKey},
		{Name: "Random", Value: base64.RawURLEncoding.EncodeToString(random)},
		{Name: "Contract-Tx-Id", Value: interaction.ContractId},
		{Name: "Tx-Id", Value: interaction.InteractionId.Base64()},
	}
	// Set previous sort key only if present
	if interaction.LastSortKey.Status == pgtype.Present {
		out = append(out, bundlr.Tag{
			Name:  "Prev-Sort-Key",
			Value: interaction.LastSortKey.String,
		})
	}

	return
}
