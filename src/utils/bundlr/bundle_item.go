package bundlr

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"strconv"
	"syncer/src/utils/arweave"
)

type BundleItem struct {
	Signature arweave.Base64String `json:"signature"`
	Owner     arweave.Base64String `json:"owner"`  //  utils.Base64Encode(pubkey)
	Target    arweave.Base64String `json:"target"` // optional, if exist must length 32, and is base64 str
	Anchor    arweave.Base64String `json:"anchor"` // optional, if exist must length 32, and is base64 str
	Tags      Tags                 `json:"tags"`
	Data      arweave.Base64String `json:"data"`
	Id        arweave.Base64String `json:"id"`
}

// Arweave signature
const SIGNATURE_LENGHT = 512
const OWNER_LENGTH = 512

func (self *BundleItem) sign(privateKey *rsa.PrivateKey, tagsBytes []byte) (id, signature []byte, err error) {
	values := []any{
		"dataitem",
		"1",
		[]byte(strconv.Itoa(1)), // Signature type
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
	self.Owner = signer.Owner

	// Signs bundle item
	self.Id, self.Signature, err = self.sign(signer.PrivateKey, tagsBytes)
	if err != nil {
		return
	}

	out = bytes.NewBuffer(make([]byte,
		0,
		2+SIGNATURE_LENGHT+OWNER_LENGTH+1+len(self.Target)+1+len(self.Anchor)+len(self.Data),
	))
	out.Write(ShortTo2ByteArray(1))
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

	out.Write(LongTo8ByteArray(len(self.Tags)))
	out.Write(LongTo8ByteArray(len(tagsBytes)))
	out.Write(tagsBytes)
	out.Write(self.Data)

	return
}
