package arweave

import "errors"

var (
	ErrFailedToParse = errors.New("failed to parse response")
	ErrBadResponse   = errors.New("bad response")
)
