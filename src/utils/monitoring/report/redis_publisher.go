package report

import (
	"go.uber.org/atomic"
)

type RedisPublisherErrors struct {
	Publish           atomic.Uint64 `json:"publish"`
	PersistentFailure atomic.Uint64 `json:"persistent"`
}

type RedisPublisherState struct {
	LastSuccessfulMessageTimestamp atomic.Int64  `json:"last_successful_message_timestamp"`
	MessagesPublished              atomic.Uint64 `json:"messages_published"`
}

type RedisPublisherReport struct {
	State  RedisPublisherState  `json:"state"`
	Errors RedisPublisherErrors `json:"errors"`
}
