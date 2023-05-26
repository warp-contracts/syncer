package bundlr

import (
	"crypto/rsa"
	"errors"

	"github.com/lestrrat-go/jwx/jwk"
)

type Signer struct {
	PrivateKey *rsa.PrivateKey
	Owner      []byte
}

func NewSigner(privateKeyJWK string) (self *Signer, err error) {
	// Parse the private key
	self = new(Signer)
	set, err := jwk.Parse([]byte(privateKeyJWK))
	if err != nil {
		return
	}
	if set.Len() != 1 {
		err = errors.New("too many keys in signer's wallet")
		return
	}

	key, ok := set.Get(0)
	if !ok {
		err = errors.New("cannot access key in JWK")
		return
	}

	var rawkey interface{}
	err = key.Raw(&rawkey)
	if err != nil {
		return
	}

	self.PrivateKey, ok = rawkey.(*rsa.PrivateKey)
	if !ok {
		err = errors.New("private ")
		return
	}

	self.Owner = self.PrivateKey.PublicKey.N.Bytes()

	return
}

func (self *Signer) GetType() SignatureType {
	return SignatureTypeArweave
}
