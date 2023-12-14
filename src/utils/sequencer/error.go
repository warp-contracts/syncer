package sequencer

import "errors"

var (
	ErrFailedToParse  = errors.New("failed to parse response")
	ErrBufferTooSmall = errors.New("buffer too small")
)
