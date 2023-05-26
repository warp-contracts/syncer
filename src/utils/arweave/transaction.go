package arweave

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"errors"
	"fmt"
	"math/big"
)

type Transaction struct {
	Format    int          `json:"format"`
	ID        string       `json:"id"`
	LastTx    Base64String `json:"last_tx"`
	Owner     Base64String `json:"owner"` // utils.Base64Encode(wallet.PubKey.N.Bytes())
	Tags      []Tag        `json:"tags"`
	Target    Base64String `json:"target"`
	Quantity  string       `json:"quantity"`
	Data      Base64String `json:"data"`
	DataSize  BigInt       `json:"data_size"`
	DataRoot  Base64String `json:"data_root"`
	Reward    string       `json:"reward"`
	Signature Base64String `json:"signature"`

	// Computed when needed.
	Chunks *Chunks `json:"-"`
}

type Tag struct {
	Name  Base64String `json:"name"`
	Value Base64String `json:"value"`
}

func (self *Transaction) GetTag(name string) (value string, ok bool) {
	for _, tag := range self.Tags {
		if string(tag.Name) == name {
			return string(tag.Value), true
		}
	}
	return
}

// https://docs.arweave.org/developers/server/http-api#transaction-signing
// Transaction signatures are generated by computing a merkle root of the SHA-384 hashes of transaction fields:
// format, owner, target, data_root, data_size, quantity, reward, last_tx, tags, then signing the hash.
// Signatures are RSA-PSS with SHA-256 as the hashing function.
func (self *Transaction) Verify() (err error) {
	if self.Format != 2 {
		err = errors.New("unsupported transaction format version")
		return
	}

	tags := make([]interface{}, 0, len(self.Tags))
	for _, tag := range self.Tags {
		tags = append(tags, []interface{}{
			[]byte(tag.Name), []byte(tag.Value),
		})
	}

	// Initialize leaves of the merkle tree
	values := []interface{}{
		fmt.Sprintf("%d", self.Format),
		self.Owner,
		self.Target,
		self.Quantity,
		self.Reward,
		self.LastTx,
		tags,
		self.DataSize,
		self.DataRoot,
	}

	ownerPublicKey := &rsa.PublicKey{
		N: new(big.Int).SetBytes([]byte(self.Owner)),
		E: 65537, //"AQAB"
	}

	deepHash := DeepHash(values)
	hash := sha256.Sum256(deepHash[:])

	return rsa.VerifyPSS(ownerPublicKey, crypto.SHA256, hash[:], []byte(self.Signature), &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
		Hash:       crypto.SHA256,
	})
}
