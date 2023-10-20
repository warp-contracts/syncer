package warp

import (
	"crypto/sha256"
	"encoding/json"

	"github.com/jackc/pgtype"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/bundlr"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/smartweave"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/sirupsen/logrus"
)

type DataItemParser struct {
	log *logrus.Entry
}

func NewDataItemParser(config *config.Config) (self *DataItemParser) {
	self = new(DataItemParser)
	self.log = logger.NewSublogger("interaction-parser")
	return
}

func (self *DataItemParser) Parse(tx *bundlr.BundleItem, blockHeight int64, blockId arweave.Base64String, blockTimestamp int64, sortKey, lastSortKey string) (out *model.Interaction, err error) {
	out = &model.Interaction{
		InteractionId:      tx.Id,
		BlockHeight:        blockHeight,
		BlockId:            blockId,
		ConfirmationStatus: "confirmed",

		// FIXME: This is a placeholder name for testing
		Source:         "sequencer-l2",
		BlockTimestamp: blockTimestamp,
		SortKey:        sortKey,
		LastSortKey:    pgtype.Text{String: lastSortKey, Status: pgtype.Present},
	}

	// Fill tags, already decoded from base64
	err = self.fillTags(tx, out)
	if err != nil {
		return
	}

	out.Owner, err = GetWalletAddressFromBundle(tx)
	if err != nil {
		return
	}

	// Save decoded tags
	parsedTags := self.parseTags(tx.Tags)

	swInteraction := smartweave.Interaction{
		Id: tx.Id,
		Owner: smartweave.Owner{
			Address: out.Owner,
		},
		Recipient: tx.Target,
		Tags:      parsedTags,
		Block: smartweave.Block{
			Height:    blockHeight,
			Id:        blockId,
			Timestamp: blockTimestamp,
		},
		// Fee: smartweave.Amount{
		// 	Winston: tx.Reward,
		// },
		// Quantity: smartweave.Amount{
		// 	Winston: tx.Quantity,
		// },
	}

	swInteractionJson, err := json.Marshal(swInteraction)
	if err != nil {
		self.log.Error("Failed to marshal interaction")
		return
	}
	err = out.Interaction.Set(swInteractionJson)
	if err != nil {
		self.log.Error("Failed set interaction JSON")
		return
	}
	return
}

func GetWalletAddressFromBundle(tx *bundlr.BundleItem) (owner string, err error) {
	// The n value is the public modulus and is used as the transaction owner field,
	// and the address of a wallet is a Base64URL encoded SHA-256 hash of the n value from the JWK.
	// https://docs.arweave.org/developers/server/http-api#addressing
	h := sha256.New()
	h.Write([]byte(tx.Owner))
	owner = base64url.Encode(h.Sum(nil))
	return
}

func (self *DataItemParser) parseTags(tags []bundlr.Tag) (out []smartweave.Tag) {
	out = make([]smartweave.Tag, len(tags))
	for i, t := range tags {
		out[i] = smartweave.Tag{
			Name:  string(t.Name),
			Value: string(t.Value),
		}
	}
	return
}

func (self *DataItemParser) fillTags(tx *bundlr.BundleItem, out *model.Interaction) (err error) {
	// Fill data from tags
	for _, t := range tx.Tags {
		err = AddTagToInteraction(out, string(t.Name), string(t.Value))
		if err != nil {
			return
		}
	}

	return nil
}
