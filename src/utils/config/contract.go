package config

import (
	"time"

	"github.com/spf13/viper"
)

type Contract struct {
	// Worker pool for fetching contact source and init state
	LoaderWorkerPoolSize int

	// Maksimum payloads in loader's queue
	LoaderWorkerQueueSize int

	// Possible contract source content types
	LoaderSupportedContentTypes []string

	// Max time for a transaction to be downloaded. 0 means no limit
	TransactionMaxElapsedTime time.Duration

	// Max time between transaction download retries
	TransactionMaxInterval time.Duration

	// How many contracts are saved in one transaction
	StoreBatchSize int

	// How often is an insert triggered
	StoreInterval time.Duration
}

func setContractDefaults() {
	viper.SetDefault("Contract.LoaderWorkerPoolSize", "50")
	viper.SetDefault("Contract.LoaderWorkerQueueSize", "100")
	viper.SetDefault("Contract.LoaderSupportedContentTypes", []string{"application/javascript", "application/wasm"})
	viper.SetDefault("Contract.TransactionMaxElapsedTime", "5m")
	viper.SetDefault("Contract.TransactionMaxInterval", "15s")
	viper.SetDefault("Contract.StoreBatchSize", "10")
	viper.SetDefault("Contract.StoreInterval", "1s")
}
