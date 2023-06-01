package config

import "github.com/spf13/viper"

type Forwarder struct {
	// How many L1 interactions are fetched from the DB at once
	FetcherBatchSize int
}

func setForwarderDefaults() {
	viper.SetDefault("Forwarder.FetcherBatchSize", "100")
}
