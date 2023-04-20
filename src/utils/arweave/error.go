package arweave

import "errors"

var (
	ErrFailedToParse = errors.New("failed to parse response")
	ErrBadResponse   = errors.New("bad response")
	ErrNotFound      = errors.New("data not found")
	ErrPending       = errors.New("tx is pending")
)

type Error struct {
	Error string `json:"error"`
}
