package warp

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgtype"
	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/config"
	"github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/utils/model"
	"github.com/warp-contracts/syncer/src/utils/smartweave"
	"github.com/warp-contracts/syncer/src/utils/tool"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/sirupsen/logrus"
)

type InteractionParser struct {
	log *logrus.Entry
}

func NewInteractionParser(config *config.Config) (self *InteractionParser, err error) {
	self = new(InteractionParser)
	self.log = logger.NewSublogger("interaction-parser")
	return
}

func (self *InteractionParser) Parse(
	tx *arweave.Transaction,
	blockHeight int64,
	blockId arweave.Base64String,
	blockTimestamp int64,
	randomValue arweave.Base64String,
	sortKey string,
	prevSortKey string,
	sequencerBlock *smartweave.SequencerBlock) (out *model.Interaction, err error) {
	out = &model.Interaction{
		InteractionId:      tx.ID,
		BlockHeight:        blockHeight,
		BlockId:            blockId,
		ConfirmationStatus: "confirmed",
		Source:             "arweave",
		BlockTimestamp:     blockTimestamp,
	}

	// Check tags format
	err = ValidateTags(tx.Tags)
	if err != nil {
		return
	}

	if prevSortKey != "" {
		err = out.LastSortKey.Set(prevSortKey)
		if err != nil {
			return
		}
	} else {
		out.LastSortKey.Status = pgtype.Null
	}

	// Fill tags, already decoded from base64
	err = self.fillTags(tx, out)
	if err != nil {
		return
	}

	if sortKey != "" {
		out.SortKey = sortKey
	} else {
		out.SortKey = CreateSortKey(tx.ID.Bytes(), blockHeight, blockId)
	}

	out.Owner, err = GetWalletAddress(tx)
	if err != nil {
		return
	}

	// Save decoded tags
	parsedTags := self.parseTags(tx.Tags)

	// Get owner's wallet address
	swInteraction := smartweave.Interaction{
		Id: tx.ID,
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
		Fee: smartweave.Amount{
			Winston: tx.Reward,
		},
		Quantity: smartweave.Amount{
			Winston: tx.Quantity,
		},
		Random: randomValue,
		SequencerBlock: sequencerBlock,
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

func GetWalletAddress(tx *arweave.Transaction) (owner string, err error) {
	// The n value is the public modulus and is used as the transaction owner field,
	// and the address of a wallet is a Base64URL encoded SHA-256 hash of the n value from the JWK.
	// https://docs.arweave.org/developers/server/http-api#addressing
	h := sha256.New()
	h.Write([]byte(tx.Owner))
	owner = base64url.Encode(h.Sum(nil))
	return
}

func (self *InteractionParser) parseTags(tags []arweave.Tag) (out []smartweave.Tag) {
	out = make([]smartweave.Tag, len(tags))
	for i, t := range tags {
		out[i] = smartweave.Tag{
			Name:  string(t.Name),
			Value: string(t.Value),
		}
	}
	return
}

func AddTagToInteraction(out *model.Interaction, name, value string) error {
	switch string(name) {
	case smartweave.TagContractTxId:
		if !arweave.TxIdRegex.MatchString(value) {
			return errors.New("tag doesn't validate as a contractId")
		}
		out.ContractId = value
	case TagInteractWrite:
		out.InteractWrite = append(out.InteractWrite, value)
	case TagWarpTestnet:
		out.Testnet = sql.NullString{
			String: value,
			Valid:  true,
		}
	case smartweave.TagInput:
		out.Input = value

		// It will allow any JSON. Arrays, objects, strings, numbers, booleans, and null
		err := tool.CheckJSON([]byte(value))
		if err != nil {
			return err
		}

		// Marshal tag into tmp struct
		var input struct {
			Function *string `json:"function"`
			Value    *string `json:"value"`
		}

		err = json.Unmarshal([]byte(out.Input), &input)
		if err != nil {
			return nil
		}

		// Check function name
		if input.Function == nil {
			return nil
		}

		// Cleanup function name
		out.Function = strings.TrimSpace(*input.Function)

		// Handle evolution
		// Is this a call to evolve
		if strings.EqualFold(out.Function, "evolve") &&
			input.Value != nil &&
			arweave.TxIdRegex.MatchString(*input.Value) {
			out.Evolve = sql.NullString{
				String: *input.Value,
				Valid:  true,
			}
		}
	}
	return nil
}

func (self *InteractionParser) fillTags(tx *arweave.Transaction, out *model.Interaction) (err error) {
	// Fill data from tags
	for _, t := range tx.Tags {
		err = AddTagToInteraction(out, string(t.Name), string(t.Value))
		if err != nil {
			return
		}
	}

	return nil
}

func CreateSortKey(transactionId []byte, blockHeight int64, blockId []byte) string {
	// Concatenate data
	buffer := make([]byte, 0, len(blockId)+len(transactionId))
	buffer = append(buffer, blockId...)
	buffer = append(buffer, transactionId...)

	// Compute hash
	sum256 := sha256.Sum256(buffer)
	hash := hex.EncodeToString(sum256[:])

	return fmt.Sprintf("%.12d,0000000000000,%s", blockHeight, hash)
}
