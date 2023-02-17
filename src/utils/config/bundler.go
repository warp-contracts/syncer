package config

import (
	"time"

	"github.com/spf13/viper"
)

type Bundler struct {
	// How often to poll the database
	PollerInterval time.Duration

	// How long does it wait for the query response
	PollerTimeout time.Duration

	// Maksimum number of requests run in parallel
	PollerMaxParallelQueries int

	// Maksimum number of interactions selected from the database in one db transaction
	PollerMaxDownloadedBatchSize int

	// Maksimum number of interactions updated in the database in one db transaction
	ConfirmerMaxBatchSize int

	// Number of workers that send bundles in parallel
	BundlerNumBundlingWorkers uint16
}

func setBundlerDefaults() {
	viper.SetDefault("Bundler.PollerInterval", "10s")
	viper.SetDefault("Bundler.PollerTimeout", "90s")
	viper.SetDefault("Bundler.PollerMaxParallelQueries", "5")
	viper.SetDefault("Bundler.PollerMaxDownloadedBatchSize", "100")
	viper.SetDefault("Bundler.ConfirmerMaxBatchSize", "1000")
	viper.SetDefault("Bundler.BundlerNumBundlingWorkers", "2")
}
