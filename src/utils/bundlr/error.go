package bundlr

import "errors"

var (
	ErrFailedToParse = errors.New("failed to parse response")
	ErrIdEmpty       = errors.New("bundle id is empty")
)
