package bundlr

import (
	"bytes"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/binary"
	"io"
	"math/big"

	"github.com/warp-contracts/syncer/src/utils/arweave"
	"github.com/warp-contracts/syncer/src/utils/tool"

	etherum_crypto "github.com/ethereum/go-ethereum/crypto"
)

type BundleItem struct {
	SignatureType SignatureType        `json:"signature_type"`
	Signature     arweave.Base64String `json:"signature"`
	Owner         arweave.Base64String `json:"owner"`  //  utils.Base64Encode(pubkey)
	Target        arweave.Base64String `json:"target"` // optional, if exist must length 32, and is base64 str
	Anchor        arweave.Base64String `json:"anchor"` // optional, if exist must length 32, and is base64 str
	Tags          Tags                 `json:"tags"`
	Data          arweave.Base64String `json:"data"`
	Id            arweave.Base64String `json:"id"`

	// Not in the standard, used internally
	tagsBytes []byte
}

// Arweave signature
const ARWEAVE_SIGNATURE_LENGHT = 512
const ARWEAVE_OWNER_LENGTH = 512

var CONFIG = map[SignatureType]struct {
	Signature int
	Owner     int
	Verify    func(hash []byte, self *BundleItem) error
}{
	SignatureTypeArweave: {
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
	SignatureTypeEtherum: {
		Signature: 65,
		Owner:     65,
		Verify: func(hash []byte, self *BundleItem) (err error) {
			// Convert owner to public key bytes
			publicKeyECDSA, err := etherum_crypto.UnmarshalPubkey(self.Owner)
			if err != nil {
				err = ErrUnmarshalEtherumPubKey
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
				err = ErrEtherumSignatureMismatch
				return
			}

			return
		},
	},
}

func (self *BundleItem) ensureTagsSerialized() (err error) {
	if len(self.tagsBytes) != 0 || len(self.Tags) == 0 {
		return nil
	}
	self.tagsBytes, err = self.Tags.Marshal()
	if err != nil {
		return err
	}
	return nil
}

func (self *BundleItem) Size() (out int) {
	signer, err := GetSigner(self.SignatureType, nil)
	if err != nil {
		return
	}

	out = 2 + signer.GetSignatureLength() + signer.GetOwnerLength() + 1 + 1 + len(self.Data)
	if len(self.Target) > 0 {
		out += len(self.Target)
	}
	if len(self.Anchor) > 0 {
		out += len(self.Anchor)
	}

	err = self.ensureTagsSerialized()
	if err != nil {
		return -1
	}

	out += len(self.tagsBytes)

	return
}

func (self *BundleItem) MarshalTo(buf []byte) (n int, err error) {
	if len(buf) < self.Size() {
		return 0, ErrBufferTooSmall
	}

	// NOTE: Normally bytes.Buffer takes ownership of the buf but in this case when we know it's big enough we ensure it won't get reallocated
	writer := bytes.NewBuffer(buf)
	err = self.Encode(nil, writer)
	if err != nil {
		return
	}

	return self.Size(), nil
}

func (self *BundleItem) sign(signer Signer) (id, signature []byte, err error) {
	values := []any{
		"dataitem",
		"1",
		self.SignatureType.Bytes(),
		self.Owner,
		self.Target,
		self.Anchor,
		self.tagsBytes,
		self.Data,
	}

	deepHash := arweave.DeepHash(values)
	hashed := sha256.Sum256(deepHash[:])

	// Compute the signature
	signature, err = signer.Sign(hashed[:])
	if err != nil {
		return
	}

	// Bundle item id
	idArray := sha256.Sum256(signature)
	id = idArray[:]

	return
}

func (self *BundleItem) Reader(signer Signer) (out *bytes.Buffer, err error) {
	// Don't try to allocate more than 4kB. Buffer will grow if needed anyway.
	initSize := tool.Max(4096, self.Size())
	out = bytes.NewBuffer(make([]byte, 0, initSize))

	err = self.Encode(signer, out)
	return
}

func (self *BundleItem) Encode(signer Signer, out *bytes.Buffer) (err error) {
	// Tags
	err = self.ensureTagsSerialized()
	if err != nil {
		return
	}

	// Crypto
	if len(self.Owner) == 0 && len(self.Signature) == 0 && len(self.Id) == 0 {
		if signer == nil {
			err = ErrSignerNotSpecified
			return
		}
		self.SignatureType = signer.GetType()
		self.Owner = signer.GetOwner()

		// Signs bundle item
		self.Id, self.Signature, err = self.sign(signer)
		if err != nil {
			return
		}
	}

	// Serialization
	out.Write(ShortTo2ByteArray(int(self.SignatureType)))
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
	out.Write(LongTo8ByteArray(len(self.tagsBytes)))
	out.Write(self.tagsBytes)
	out.Write(self.Data)

	return
}

func (self *BundleItem) Unmarshal(buf []byte) (err error) {
	reader := bytes.NewReader(buf)
	return self.UnmarshalFromReader(reader)
}

// Reverse operation of Reader
func (self *BundleItem) UnmarshalFromReader(reader io.Reader) (err error) {
	// Signature type
	signatureType := make([]byte, 2)
	n, err := reader.Read(signatureType)
	if err != nil {
		return
	}
	if n < 2 {
		err = ErrNotEnoughBytesForSignatureType
		return
	}
	self.SignatureType = SignatureType(binary.LittleEndian.Uint16(signatureType))

	// For now only Arweave signature is supported
	if self.SignatureType != 1 {
		err = ErrUnsupportedSignatureType
		return
	}

	// Signature (different length depending on the signature type)
	self.Signature = make([]byte, CONFIG[self.SignatureType].Signature)
	n, err = reader.Read(self.Signature)
	if err != nil {
		return
	}
	if n < CONFIG[self.SignatureType].Signature {
		err = ErrNotEnoughBytesForSignature
		return
	}

	// Owner - public key (different length depending on the signature type)
	self.Signature = make([]byte, CONFIG[self.SignatureType].Owner)
	n, err = reader.Read(self.Signature)
	if err != nil {
		return
	}
	if n < CONFIG[self.SignatureType].Owner {
		err = ErrNotEnoughBytesForOwner
		return
	}

	// Target (it's optional)
	isTargetPresent := make([]byte, 1)
	n, err = reader.Read(isTargetPresent)
	if err != nil {
		return err
	}
	if n < 1 {
		err = ErrNotEnoughBytesForTargetFlag
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
			err = ErrNotEnoughBytesForTarget
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
		err = ErrNotEnoughBytesForAnchorFlag
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
			err = ErrNotEnoughBytesForAnchor
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
		err = ErrNotEnoughBytesForNumberOfTags
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
		err = ErrNotEnoughBytesForNumberOfTagBytes
		return
	}
	numTagsBytes := int(binary.LittleEndian.Uint64(numTagsBytesBuffer))

	// Tags
	self.Tags = make([]Tag, numTags)
	if numTags > 0 {
		// Read tags
		self.tagsBytes = make([]byte, numTagsBytes)
		n, err = reader.Read(self.tagsBytes)
		if err != nil {
			return
		}
		if n < numTagsBytes {
			err = ErrNotEnoughBytesForTags
			return
		}

		// Parse tags
		err = self.Tags.Unmarshal(self.tagsBytes)
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
		err = ErrVerifyIdSignatureMismatch
		return
	}

	// an anchor isn't more than 32 bytes
	// with this lib it has to be 0 or 32bytes
	if len(self.Anchor) != 0 && len(self.Anchor) != 32 {
		err = ErrVerifyBadAnchorLength
		return
	}

	// Tags
	if len(self.Tags) > 128 {
		err = ErrVerifyTooManyTags
		return
	}

	for _, tag := range self.Tags {
		if len(tag.Name) == 0 {
			err = ErrVerifyEmptyTagName
			return
		}
		if len(tag.Name) > 1024 {
			err = ErrVerifyTooLongTagName
			return
		}
		if len(tag.Value) == 0 {
			err = ErrVerifyEmptyTagValue
			return
		}
		if len(tag.Value) > 3072 {
			err = ErrVerifyTooLongTagValue
			return
		}
	}

	// Verify signature
	return self.VerifySignature()
}

func (self *BundleItem) VerifySignature() (err error) {
	err = self.ensureTagsSerialized()
	if err != nil {
		return
	}

	values := []any{
		"dataitem",
		"1",
		self.SignatureType.Bytes(),
		self.Owner,
		self.Target,
		self.Anchor,
		self.tagsBytes,
		self.Data,
	}

	deepHash := arweave.DeepHash(values)
	hashed := sha256.Sum256(deepHash[:])

	signer, err := GetSigner(self.SignatureType, self.Owner)
	if err != nil {
		return
	}

	return signer.Verify(hashed[:], self.Signature)
}
