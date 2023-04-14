package tool

import (
	"bytes"
	"encoding/json"
)

func MinifyJSON(in []byte) []byte {
	dst := &bytes.Buffer{}
	if err := json.Compact(dst, in); err != nil {
		panic(err)
	}
	return dst.Bytes()
}

func IsJSON(in []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(in, &js) == nil
}
