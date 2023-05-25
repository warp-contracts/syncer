package bundlr

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"io"
	"math/big"
	"strconv"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/tool"

	etherum_crypto "github.com/ethereum/go-ethereum/crypto"
)

type BundleItem struct {
	SignatureType int                  `json:"signature_type"`
	Signature     arweave.Base64String `json:"signature"`
	Owner         arweave.Base64String `json:"owner"`  //  utils.Base64Encode(pubkey)
	Target        arweave.Base64String `json:"target"` // optional, if exist must length 32, and is base64 str
	Anchor        arweave.Base64String `json:"anchor"` // optional, if exist must length 32, and is base64 str
	Tags          Tags                 `json:"tags"`
	Data          arweave.Base64String `json:"data"`
	Id            arweave.Base64String `json:"id"`
}

// Arweave signature
const ARWEAVE_SIGNATURE_LENGHT = 512
const ARWEAVE_OWNER_LENGTH = 512

var CONFIG = map[int]struct {
	Signature int
	Owner     int
	Verify    func(hash []byte, self *BundleItem) error
}{
	1: {
		Signature: 512,
		Owner:     512,
		Verify: func(hash []byte, self *BundleItem) error {
			ownerPublicKey := &rsa.PublicKey{
				N: new(big.Int).SetBytes([]byte(self.Owner)),
				E: 65537, //"AQAB"
			}

			return rsa.VerifyPSS(ownerPublicKey, crypto.SHA256, hash, []byte(self.Signature), &rsa.PSSOptions{
				SaltLength: rsa.PSSSaltLengthAuto,
				Hash:       crypto.SHA256,
			})
		},
	},
	3: {
		Signature: 65,
		Owner:     65,
		Verify: func(hash []byte, self *BundleItem) (err error) {
			// Convert owner to public key bytes
			publicKeyECDSA, err := etherum_crypto.UnmarshalPubkey(self.Owner)
			if err != nil {
				err = errors.New("can not unmarshal etherum pubkey")
				return
			}
			publicKeyBytes := etherum_crypto.FromECDSAPub(publicKeyECDSA)

			// Get the public key from the signature
			sigPublicKey, err := etherum_crypto.Ecrecover(hash, self.Signature)
			if err != nil {
				return
			}

			// Check if the public key recovered from the signature matches the owner
			if !bytes.Equal(sigPublicKey, publicKeyBytes) {
				err = errors.New("verify etherum signature failed")
				return
			}

			return
		},
	},
}

func (self *BundleItem) sign(privateKey *rsa.PrivateKey, tagsBytes []byte) (id, signature []byte, err error) {
	values := []any{
		"dataitem",
		"1",
		[]byte(strconv.Itoa(self.SignatureType)), // Signature type
		self.Owner,
		self.Target,
		self.Anchor,
		tagsBytes,
		self.Data,
	}

	deepHash := arweave.DeepHash(values)
	hashed := sha256.Sum256(deepHash[:])

	// Compute the signature
	signature, err = rsa.SignPSS(rand.Reader, privateKey, crypto.SHA256, hashed[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthAuto,
		Hash:       crypto.SHA256,
	})
	if err != nil {
		return
	}

	// Bundle item id
	idArray := sha256.Sum256(signature)
	id = idArray[:]

	return
}

func (self *BundleItem) Reader(signer *Signer) (out *bytes.Buffer, err error) {
	// Tags
	tagsBytes, err := self.Tags.Marshal()
	if err != nil {
		return
	}

	// Crypto
	if len(self.Owner) == 0 && len(self.Signature) == 0 && len(self.Id) == 0 {
		self.SignatureType = signer.GetType()
		self.Owner = signer.Owner

		// Signs bundle item
		self.Id, self.Signature, err = self.sign(signer.PrivateKey, tagsBytes)
		if err != nil {
			return
		}
	}

	// Serialization
	// Don't try to allocate more than 4kB. Buffer will grow if needed anyway.
	initSize := tool.Max(4096, 2+ARWEAVE_SIGNATURE_LENGHT+ARWEAVE_OWNER_LENGTH+1+len(self.Target)+1+len(self.Anchor)+len(self.Data))
	out = bytes.NewBuffer(make([]byte, 0, initSize))

	out.Write(ShortTo2ByteArray(self.SignatureType))
	out.Write(self.Signature)
	out.Write(self.Owner)

	// Optional target
	if len(self.Target) == 0 {
		out.WriteByte(0)
	} else {
		out.WriteByte(1)
		out.Write(self.Target)
	}

	// Optional anchor
	if len(self.Anchor) == 0 {
		out.WriteByte(0)
	} else {
		out.WriteByte(1)
		out.Write(self.Anchor)
	}

	// Rest
	out.Write(LongTo8ByteArray(len(self.Tags)))
	out.Write(LongTo8ByteArray(len(tagsBytes)))
	out.Write(tagsBytes)
	out.Write(self.Data)

	return
}

// Reverse operation of Reader, but use subslices instead of copying.
// Parsed buffer is saved for later reuse.
// User should ensure data buffer is not modified AFTER calling this method
func (self *BundleItem) Unmarshal(reader io.Reader) (err error) {
	// Signature type
	signatureType := make([]byte, 2)
	n, err := reader.Read(signatureType)
	if err != nil {
		return
	}
	if n < 2 {
		err = errors.New("not enough bytes for the signature type")
		return
	}
	self.SignatureType = int(binary.LittleEndian.Uint16(signatureType))

	// For now only Arweave signature is supported
	if self.SignatureType != 1 {
		err = errors.New("only Arweave signature is supported")
		return
	}

	// Signature (different length depending on the signature type)
	self.Signature = make([]byte, CONFIG[self.SignatureType].Signature)
	n, err = reader.Read(self.Signature)
	if err != nil {
		return
	}
	if n < CONFIG[self.SignatureType].Signature {
		err = errors.New("not enough bytes for the signature")
		return
	}

	// Owner - public key (different length depending on the signature type)
	self.Signature = make([]byte, CONFIG[self.SignatureType].Owner)
	n, err = reader.Read(self.Signature)
	if err != nil {
		return
	}
	if n < CONFIG[self.SignatureType].Owner {
		err = errors.New("not enough bytes for the owner")
		return
	}

	// Target (it's optional)
	isTargetPresent := make([]byte, 1)
	n, err = reader.Read(isTargetPresent)
	if err != nil {
		return err
	}
	if n < 1 {
		err = errors.New("not enough bytes for the target flag")
		return
	}

	if isTargetPresent[0] == 0 {
		self.Target = []byte{}
	} else {
		// Value present
		self.Target = make([]byte, 32)
		n, err = reader.Read(self.Target)
		if err != nil {
			return err
		}
		if n < 32 {
			err = errors.New("not enough bytes for the target")
			return
		}
	}

	// Anchor (it's optional)
	isAnchorPresent := make([]byte, 1)
	n, err = reader.Read(isAnchorPresent)
	if err != nil {
		return err
	}
	if n < 1 {
		err = errors.New("not enough bytes for the anchor flag")
		return
	}

	if isAnchorPresent[0] == 0 {
		self.Anchor = []byte{}
	} else {
		// Value present
		self.Anchor = make([]byte, 32)
		n, err = reader.Read(self.Anchor)
		if err != nil {
			return err
		}
		if n < 32 {
			err = errors.New("not enough bytes for the anchor")
			return
		}
	}

	// Length of the tags slice
	numTagsBuffer := make([]byte, 8)
	n, err = reader.Read(numTagsBuffer)
	if err != nil {
		return
	}
	if n < 8 {
		err = errors.New("not enough bytes for the number of tags")
		return
	}
	numTags := int(binary.LittleEndian.Uint64(numTagsBuffer))

	// Size of encoded tags
	numTagsBytesBuffer := make([]byte, 8)
	n, err = reader.Read(numTagsBytesBuffer)
	if err != nil {
		return
	}
	if n < 8 {
		err = errors.New("not enough bytes for the number of bytes for tags")
		return
	}
	numTagsBytes := int(binary.LittleEndian.Uint64(numTagsBytesBuffer))

	// Tags
	self.Tags = make([]Tag, numTags)
	if numTags > 0 {
		// Read tags
		tagsBuffer := make([]byte, numTagsBytes)
		n, err = reader.Read(tagsBuffer)
		if err != nil {
			return
		}
		if n < numTagsBytes {
			err = errors.New("not enough bytes for the tags")
			return
		}

		// Parse tags
		err = self.Tags.Unmarshal(tagsBuffer)
		if err != nil {
			return
		}
	}

	// The rest is just data
	var data bytes.Buffer
	_, err = data.ReadFrom(reader)
	if err != nil {
		return
	}
	self.Data = data.Bytes()

	return
}

// https://github.com/ArweaveTeam/arweave-standards/blob/master/ans/ANS-104.md#21-verifying-a-dataitem
func (self *BundleItem) Verify() (err error) {
	idArray := sha256.Sum256(self.Signature)
	if !bytes.Equal(idArray[:], self.Id) {
		err = errors.New("id doesn't match signature")
		return
	}

	// an anchor isn't more than 32 bytes
	// with this lib it has to be 0 or 32bytes
	if len(self.Anchor) != 0 && len(self.Anchor) != 32 {
		err = errors.New("anchor must be 32 bytes long")
		return
	}

	// Tags
	if len(self.Tags) > 128 {
		err = errors.New("too many tags, max is 128")
		return
	}

	for _, tag := range self.Tags {
		if len(tag.Name) == 0 {
			err = errors.New("tag name is empty")
			return
		}
		if len(tag.Name) > 1024 {
			err = errors.New("tag name is too long, max is 1024 bytes")
			return
		}
		if len(tag.Value) == 0 {
			err = errors.New("tag value is empty")
			return
		}
		if len(tag.Value) > 3072 {
			err = errors.New("tag value is too long, max is 3072 bytes")
			return
		}
	}

	// Verify signature
	return self.VerifySignature()
}

func (self *BundleItem) VerifySignature() (err error) {
	tagsBytes, err := self.Tags.Marshal()
	if err != nil {
		return
	}

	values := []any{
		"dataitem",
		"1",
		[]byte(strconv.Itoa(self.SignatureType)),
		self.Owner,
		self.Target,
		self.Anchor,
		tagsBytes,
		self.Data,
	}

	deepHash := arweave.DeepHash(values)
	hashed := sha256.Sum256(deepHash[:])

	conf, ok := CONFIG[self.SignatureType]
	if !ok {
		err = errors.New("unknown signature type")
		return
	}

	return conf.Verify(hashed[:], self)
}
