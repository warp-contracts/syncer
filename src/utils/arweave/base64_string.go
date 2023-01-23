package arweave

import (
	"encoding/base64"
	"encoding/json"
)

type Base64String []byte

func (self *Base64String) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	// Decode base64
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	*self = []byte(b)
	return nil
}

func (self *Base64String) MarshalJSON() (out []byte, err error) {
	s := base64.RawURLEncoding.EncodeToString([]byte(*self))
	return json.Marshal(s)
}
