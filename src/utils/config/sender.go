package config

import (
	"time"

	"github.com/spf13/viper"
)

type Sender struct {
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

	// How often notifier monitors the number of pending tx to bundle
	DBPollerInterval time.Duration

	// Switch off listening for async notifications
	NotifierDisabled bool

	// Maksimum number of workers that handle notifications
	NotifierWorkerPoolSize int

	// Maksimum notifications waiting in the queue
	NotifierWorkerQueueSize int

	// How many batches are confirmet in one transaction
	StoreBatchSize int

	// How often are states updated in the database
	StoreInterval time.Duration

	// Max time store will try to insert a batch of data to the database
	// 0 means no limit
	// This should be 0,
	StoreBackoffMaxElapsedTime time.Duration

	// Max time between retries to insert a batch of confirmations to  the database
	StoreBackoffMaxInterval time.Duration

	// Number of workers that send bundles in parallel
	BundlerNumBundlingWorkers int
}

func setSenderDefaults() {
	viper.SetDefault("Sender.PollerDisabled", "false")
	viper.SetDefault("Sender.PollerInterval", "10s")
	viper.SetDefault("Sender.PollerTimeout", "90s")
	viper.SetDefault("Sender.PollerMaxParallelQueries", "50")
	viper.SetDefault("Sender.PollerMaxDownloadedBatchSize", "100")
	viper.SetDefault("Sender.PollerMaxBatchSize", "100")
	viper.SetDefault("Sender.PollerRetryBundleAfter", "60m")
	viper.SetDefault("Sender.DBPollerInterval", "5m")
	viper.SetDefault("Sender.WorkerPoolQueueSize", "5")
	viper.SetDefault("Sender.BundlerNumBundlingWorkers", "50")
	viper.SetDefault("Sender.NotifierDisabled", "false")
	viper.SetDefault("Sender.NotifierWorkerPoolSize", "50")
	viper.SetDefault("Sender.NotifierWorkerQueueSize", "100")
	viper.SetDefault("Sender.StoreBatchSize", "100")
	viper.SetDefault("Sender.StoreInterval", "1s")
	viper.SetDefault("Sender.StoreBackoffMaxElapsedTime", "0")
	viper.SetDefault("Sender.StoreBackoffMaxInterval", "8s")
}
