package bundlr

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"

	"github.com/ethereum/go-ethereum/common/hexutil"
	etherum_crypto "github.com/ethereum/go-ethereum/crypto"
)

type EtherumSigner struct {
	PrivateKey *ecdsa.PrivateKey
	Owner      []byte
}

func NewEtherumSigner(privateKeyHex string) (self *EtherumSigner, err error) {
	self = new(EtherumSigner)

	// Parse the private key
	buf, err := hexutil.Decode(privateKeyHex)
	if err != nil {
		return
	}

	self.PrivateKey, err = etherum_crypto.ToECDSA(buf)
	if err != nil {
		return
	}

	return
}

func (self *EtherumSigner) Sign(data []byte) (signature []byte, err error) {
	hashed := sha256.Sum256(data)
	return etherum_crypto.Sign(hashed[:], self.PrivateKey)
}

func (self *EtherumSigner) Verify(data []byte, signature []byte) (err error) {
	hashed := sha256.Sum256(data)

	if len(self.Owner) == 0 {
		self.Owner = self.GetOwner()
	}

	// Convert owner to public key bytes
	publicKeyECDSA, err := etherum_crypto.UnmarshalPubkey(self.Owner)
	if err != nil {
		err = ErrUnmarshalEtherumPubKey
		return
	}
	publicKeyBytes := etherum_crypto.FromECDSAPub(publicKeyECDSA)

	// Get the public key from the signature
	sigPublicKey, err := etherum_crypto.Ecrecover(hashed[:], signature)
	if err != nil {
		return
	}

	// Check if the public key recovered from the signature matches the owner
	if !bytes.Equal(sigPublicKey, publicKeyBytes) {
		err = ErrEtherumSignatureMismatch
		return
	}

	return
}

func (self *EtherumSigner) GetOwner() []byte {
	publicKeyECDSA, ok := self.PrivateKey.Public().(*ecdsa.PublicKey)
	if !ok {
		panic(ErrFailedToParseEtherumPublicKey)
	}

	return etherum_crypto.FromECDSAPub(publicKeyECDSA)

}

func (self *EtherumSigner) GetType() SignatureType {
	return SignatureTypeEtherum
}

func (self *EtherumSigner) GetSignatureLength() int {
	return 65
}

func (self *EtherumSigner) GetOwnerLength() int {
	return 65
}
