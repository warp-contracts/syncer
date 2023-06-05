package config

import "github.com/spf13/viper"

type Forwarder struct {
	// How many L1 interactions are fetched from the DB at once
	FetcherBatchSize int

	// Interactions are saved to this Redis channel
	PublisherRedisChannelName string
}

func setForwarderDefaults() {
	viper.SetDefault("Forwarder.FetcherBatchSize", "100")
	viper.SetDefault("Forwarder.PublisherRedisChannelName", "interactions")
}
