package arweave

import (
	"encoding/base64"
	"encoding/json"
)

type RewardAddr []byte

func (self *RewardAddr) UnmarshalJSON(data []byte) error {
	var s string
	err := json.Unmarshal(data, &s)
	if err != nil {
		return err
	}

	if s == "unclaimed" {
		*self = []byte("unclaimed")
		return nil
	}

	// Decode base64
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return err
	}

	*self = []byte(b)
	return nil
}
