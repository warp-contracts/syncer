package bundlr

import "strconv"

type SignatureType int

const (
	SignatureTypeArweave SignatureType = 1
	SignatureTypeEtherum SignatureType = 3
)

func (self SignatureType) Bytes() []byte {
	return []byte(strconv.Itoa(int(self)))
}
