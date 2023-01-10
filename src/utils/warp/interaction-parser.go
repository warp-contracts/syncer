package warp

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"syncer/src/utils/arweave"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syncer/src/utils/model"
	"syncer/src/utils/smartweave"

	"github.com/dvsekhvalnov/jose2go/base64url"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/sirupsen/logrus"
)

type InteractionParser struct {
	jwkKey jwk.Key
	log    *logrus.Entry
}

var (
	contractIdRegex = regexp.MustCompile("^[a-zA-Z0-9_-]{43}$")
	txIdRegex       = regexp.MustCompile("^[a-z0-9_-]{43}$")
)

func NewInteractionParser(config *config.Config) (self *InteractionParser, err error) {
	self = new(InteractionParser)
	self.log = logger.NewSublogger("interaction-parser")
	self.jwkKey, err = jwk.ParseKey([]byte(config.ArWalletJwk))
	return
}

func (self *InteractionParser) Parse(tx *arweave.Transaction, blockHeight int64, blockId string, blockTimestamp int64) (out *model.Interaction, err error) {
	out = &model.Interaction{
		InteractionId:      tx.ID,
		BlockHeight:        blockHeight,
		BlockId:            blockId,
		ConfirmationStatus: "confirmed",
		Source:             "arweave",
		// Owner:              tx.Owner,
	}

	// Fill tags, already decoded from base64
	err = self.fillTags(tx, out)
	if err != nil {
		return
	}

	out.SortKey = self.createSortKey(tx, blockHeight, blockId)

	out.Owner, err = self.getOwner(tx)
	if err != nil {
		return
	}

	// Get owner's wallet address
	swInteraction := smartweave.Interaction{
		Id: tx.ID,
		Owner: smartweave.Owner{
			Address: tx.Owner,
		},
		Recipient: tx.Target,
		Tags:      tx.Tags,
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
	}

	swInteractionJson, err := json.Marshal(swInteraction)
	if err != nil {
		self.log.Error("Failed to marshal interaction")
		return
	}
	out.Interaction = string(swInteractionJson)

	return
}

func (self *InteractionParser) getOwner(tx *arweave.Transaction) (owner string, err error) {
	// The n value is the public modulus and is used as the transaction owner field,
	// and the address of a wallet is a Base64URL encoded SHA-256 hash of the n value from the JWK.
	// https://docs.arweave.org/developers/server/http-api#addressing
	n, err := base64url.Decode(tx.Owner)
	if err != nil {
		self.log.WithError(err).Error("Failed to decode owner")
		return
	}

	h := sha256.New()
	h.Write(n)
	owner = base64url.Encode(h.Sum(nil))

	return
}

func (self *InteractionParser) fillTags(tx *arweave.Transaction, out *model.Interaction) (err error) {
	// Fill data from tags
	var value string
	for _, t := range tx.Tags {
		// Base64String to string
		value = string(t.Value)
		switch t.Name {
		case "Contract":
			if !contractIdRegex.MatchString(value) {
				err = errors.New("tag doesn't validate as a contractId")
				self.log.Error("Failed to validate contract id")
				return
			}
			out.ContractId = value
		case "Interact-Write":
			out.InteractWrite = append(out.InteractWrite, value)
		case "Warp-Testnet":
			out.Testnet = sql.NullString{
				String: value,
				Valid:  true,
			}
		case "Input":
			out.Input = value

			// Marshal tag into tmp struct
			var input struct {
				Function *string `json:"function"`
				Value    *string `json:"value"`
			}

			err = json.Unmarshal([]byte(out.Input), &input)
			if err != nil {
				self.log.Error("Failed to parse function in input")
				return
			}

			// Check function name
			if input.Function == nil {
				break
			}

			// Cleanup function name
			out.Function = strings.TrimSpace(*input.Function)

			// Handle evolution
			// Is this a call to evolve
			if strings.EqualFold(out.Function, "evolve") &&
				input.Value != nil &&
				txIdRegex.MatchString(*input.Value) {
				out.Evolve = sql.NullString{
					String: *input.Value,
					Valid:  true,
				}
			}

		}
	}

	return nil
}

func (self *InteractionParser) createSortKey(tx *arweave.Transaction, blockHeight int64, id string) string {
	blockId := []byte(id)
	transactionId := []byte(tx.ID)
	d := (self.jwkKey.(jwk.RSAPrivateKey)).D()

	// Concatenate data
	buffer := make([]byte, 0, len(blockId)+len(transactionId)+len(d))
	buffer = append(buffer, blockId...)
	buffer = append(buffer, transactionId...)
	buffer = append(buffer, d...)

	// Compute hash
	sum256 := sha256.Sum256(buffer)
	hash := hex.EncodeToString(sum256[:])

	return fmt.Sprintf("%.12d,0000000000000,%s", blockHeight, hash)
}
