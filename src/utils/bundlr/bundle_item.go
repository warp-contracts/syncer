package bundlr

import (
	"bytes"
	"errors"
)

type BundleItem struct {
	Signature []byte `json:"signature"`
	Owner     []byte `json:"owner"`  //  utils.Base64Encode(pubkey)
	Target    []byte `json:"target"` // optional, if exist must length 32, and is base64 str
	Anchor    []byte `json:"anchor"` // optional, if exist must length 32, and is base64 str
	Tags      Tags   `json:"tags"`
	Data      []byte `json:"data"`
}

// Arweave signature
const SIGNATURE_LENGHT = 512
const OWNER_LENGTH = 512

func (self *BundleItem) Reader() (out *bytes.Buffer, err error) {
	if len(self.Signature) != SIGNATURE_LENGHT {
		err = errors.New("bad signature length")
		return
	}

	if len(self.Owner) != OWNER_LENGTH {
		err = errors.New("signature length incorrect")
		return
	}

	out = bytes.NewBuffer(make([]byte,
		0,
		2+SIGNATURE_LENGHT+OWNER_LENGTH+1+len(self.Target)+1+len(self.Anchor),
	))
	out.Write(ShortTo2ByteArray(1))
	out.Write(self.Signature)
	out.Write(self.Owner)

	if len(self.Target) == 0 {
		out.WriteByte(0)
	} else {
		out.WriteByte(1)
		out.Write(self.Target)
	}

	if len(self.Anchor) == 0 {
		out.WriteByte(0)
	} else {
		out.WriteByte(1)
		out.Write(self.Anchor)
	}

	// Tags
	tagsBytes, err := self.Tags.Marshal()
	if err != nil {
		return
	}
	out.Write(LongTo8ByteArray(len(self.Tags)))
	out.Write(LongTo8ByteArray(len(tagsBytes)))
	out.Write(tagsBytes)
	out.Write(self.Data)

	return
}
