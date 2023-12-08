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

func setEvolveDefaults() {
	viper.SetDefault("Evolve.PollerInterval", "20s")
	viper.SetDefault("Evolve.PollerTimeout", "90s")
	viper.SetDefault("Evolve.PollerChannelBufferLength", 100)
	viper.SetDefault("Evolve.PollerMaxBatchSize", 100)
	viper.SetDefault("Evolve.DownloaderSourceTransactiondMaxInterval", "5s")
	viper.SetDefault("Evolve.DownloaderNumWorkers", "50")
	viper.SetDefault("Evolve.DownloaderWorkerQueueSize", "10")
	viper.SetDefault("Evolve.StoreBatchSize", "10")
	viper.SetDefault("Evolve.StoreInterval", "10s")
	viper.SetDefault("Evolve.StoreBackoffMaxElapsedTime", "0")
}
