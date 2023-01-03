package sortkey

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/everFinance/goar/utils"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

func CreateSortKey(
	jwkKey jwk.Key,
	blockId []byte,
	mills int64,
	transactionId []byte,
	blockHeight int64,
) (string, error) {
	d := (jwkKey.(jwk.RSAPrivateKey)).D()
	buffer := utils.ConcatBuffer(blockId, transactionId, d)
	sum256 := sha256.Sum256(buffer)
	hash := hex.EncodeToString(sum256[:])
	blockHeightString := fmt.Sprintf("%.12d", blockHeight)
	return fmt.Sprintf("%s,%d,%s", blockHeightString, mills, hash), nil
}
