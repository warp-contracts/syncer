package config

import (
	"time"

	"github.com/spf13/viper"
)

type Forwarder struct {
	// How many L1 interactions are fetched from the DB at once
	FetcherBatchSize int

	// Interactions are saved to this Redis channel
	PublisherRedisChannelName string

	// How long to wait before after receiving a new block height before sending L1 interactions
	// This delay ensures sequencer finishes handling requests in time
	HeightDelay time.Duration
}

func setForwarderDefaults() {
	viper.SetDefault("Forwarder.FetcherBatchSize", "10")
	viper.SetDefault("Forwarder.PublisherRedisChannelName", "interactions")
	viper.SetDefault("Forwarder.HeightDelay", "1s")
}
