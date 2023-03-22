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

	// Max queries in the queue
	WorkerPoolQueueSize int

	// Maksimum number of interactions selected from the database in one db transaction
	PollerMaxDownloadedBatchSize int

	// Maksimum number of workers that handle notifications
	NotifierWorkerPoolSize int

	// Maksimum notifications waiting in the queue
	NotifierWorkerQueueSize int

	// Maksimum number of interactions updated in the database in one db transaction
	ConfirmerMaxBatchSize int

	// Retry sending bundles to bundlr.network
	ConfirmerRetryBundleAfter time.Duration

	// Number of workers that send bundles in parallel
	BundlerNumBundlingWorkers uint16
}

func setBundlerDefaults() {
	viper.SetDefault("Bundler.PollerInterval", "10s")
	viper.SetDefault("Bundler.PollerTimeout", "90s")
	viper.SetDefault("Bundler.PollerMaxParallelQueries", "50")
	viper.SetDefault("Bundler.WorkerPoolQueueSize", "1000")
	viper.SetDefault("Bundler.PollerMaxDownloadedBatchSize", "100")
	viper.SetDefault("Bundler.ConfirmerMaxBatchSize", "1000")
	viper.SetDefault("Bundler.ConfirmerRetryBundleAfter", "10m")
	viper.SetDefault("Bundler.BundlerNumBundlingWorkers", "2")
	viper.SetDefault("Bundler.NotifierWorkerPoolSize", "5")
	viper.SetDefault("Bundler.NotifierWorkerQueueSize", "100")
}
