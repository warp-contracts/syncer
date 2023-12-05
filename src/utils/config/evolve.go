package config

import (
	"time"

	"github.com/spf13/viper"
)

type Evolve struct {
	// How often to poll the database
	PollerInterval time.Duration


	// How long does it wait for the query response
	PollerTimeout time.Duration

	// Maximum length of the channel buffer
	ChannelBufferLength int

	// Maximum number of evolved contract sources to be updated in the database
	PollerMaxBatchSize int
}

func setEvolveDefaults() {
	viper.SetDefault("Evolve.PollerInterval", "10s")
	viper.SetDefault("Evolve.PollerTimeout", "90s")
	viper.SetDefault("Evolve.ChannelBufferLength", 100)
	viper.SetDefault("Evolve.PollerMaxBatchSize", 100)
}
