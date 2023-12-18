package config

import (
	"time"

	"github.com/spf13/viper"
)

type Evolver struct {
	// How often to poll the database
	PollerInterval time.Duration


	// How long does it wait for the query response
	PollerTimeout time.Duration

	// Maximum length of the channel buffer
	PollerChannelBufferLength int

	// Maximum number of evolved contract sources to be updated in the database
	PollerMaxBatchSize int

	// Max time between evolved source transaction retry
	DownloaderSourceTransactiondMaxInterval time.Duration

	// Number of workers that download the transactions
	DownloaderNumWorkers int
	
	// Max number of transactions that wait in the worker queue
	DownloaderWorkerQueueSize int

	// How many contract sources are saved in one transaction
	StoreBatchSize int

	// How often is an insert triggered
	StoreInterval time.Duration

	// Max time store will try to insert a batch of data to the database
	// 0 means no limit
	// This should be 0,
	StoreBackoffMaxElapsedTime time.Duration

	// Max time between retries to insert a batch of confirmations to  the database
	StoreBackoffMaxInterval time.Duration
}

func setEvolverDefaults() {
	viper.SetDefault("Evolver.PollerInterval", "1m")
	viper.SetDefault("Evolver.PollerTimeout", "90s")
	viper.SetDefault("Evolver.PollerChannelBufferLength", 100)
	viper.SetDefault("Evolver.PollerMaxBatchSize", 100)
	viper.SetDefault("Evolver.DownloaderSourceTransactiondMaxInterval", "5s")
	viper.SetDefault("Evolver.DownloaderNumWorkers", "50")
	viper.SetDefault("Evolver.DownloaderWorkerQueueSize", "10")
	viper.SetDefault("Evolver.StoreBatchSize", "10")
	viper.SetDefault("Evolver.StoreInterval", "10s")
	viper.SetDefault("Evolver.StoreBackoffMaxElapsedTime", "0")
	viper.SetDefault("Evolver.StoreBackoffMaxInterval", "20s")
}
