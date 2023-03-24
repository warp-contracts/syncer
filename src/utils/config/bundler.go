package config

import (
	"time"

	"github.com/spf13/viper"
)

type Bundler struct {
	// Disable polling mechanism
	PollerDisabled bool

	// How often to poll the database
	PollerInterval time.Duration

	// How long does it wait for the query response
	PollerTimeout time.Duration

	// Maksimum number of requests run in parallel
	PollerMaxParallelQueries int

	// Maksimum number of interactions updated in the database in one db transaction
	PollerMaxBatchSize int

	// Retry sending bundles to bundlr.network
	PollerRetryBundleAfter time.Duration

	// Max queries in the queue
	WorkerPoolQueueSize int

	// Maksimum number of interactions selected from the database in one db transaction
	PollerMaxDownloadedBatchSize int

	// Switch off listening for async notifications
	NotifierDisabled bool

	// Maksimum number of workers that handle notifications
	NotifierWorkerPoolSize int

	// Maksimum notifications waiting in the queue
	NotifierWorkerQueueSize int

	// How many batches are confirmet in one transaction
	ConfirmerBatchSize int

	// How often are states updated in the database
	ConfirmerInterval time.Duration

	// Number of workers that send bundles in parallel
	BundlerNumBundlingWorkers int
}

func setBundlerDefaults() {
	viper.SetDefault("Bundler.PollerDisabled", "false")
	viper.SetDefault("Bundler.PollerInterval", "10s")
	viper.SetDefault("Bundler.PollerTimeout", "90s")
	viper.SetDefault("Bundler.PollerMaxParallelQueries", "50")
	viper.SetDefault("Bundler.PollerMaxDownloadedBatchSize", "100")
	viper.SetDefault("Bundler.PollerMaxBatchSize", "100")
	viper.SetDefault("Bundler.PollerRetryBundleAfter", "10m")
	viper.SetDefault("Bundler.WorkerPoolQueueSize", "100")
	viper.SetDefault("Bundler.BundlerNumBundlingWorkers", "50")
	viper.SetDefault("Bundler.NotifierDisabled", "false")
	viper.SetDefault("Bundler.NotifierWorkerPoolSize", "50")
	viper.SetDefault("Bundler.NotifierWorkerQueueSize", "100")
	viper.SetDefault("Bundler.ConfirmerBatchSize", "100")
	viper.SetDefault("Bundler.ConfirmerInterval", "1s")
}
