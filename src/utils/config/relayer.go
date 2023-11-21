package config

import (
	"time"

	"github.com/spf13/viper"
)

type Relayer struct {
	// Where is the relayer started (dev, main, test)
	Environment string

	// Websocket url of the Warp's sequencer
	SequencerUrl string

	// How many incomming events should be stored in channel
	SequencerQueueSize int

	// Max time request for a block to be downloaded. 0 means no limit
	ArweaveBlockDownloadTimeout time.Duration

	// Max time for a block to be downloaded. 0 means no limit
	ArweaveBlockDownloadMaxElapsedTime time.Duration

	// Max time between transaction download retries
	ArweaveBlockDownloadMaxInterval time.Duration

	// Worker pool size for downloading Sequencer's blocks
	SourceMaxWorkers int

	// Worker pool queue size for downloading Sequencer's  blocks
	SourceMaxQueueSize int

	// Max time a request should be retried. 0 means no limit.
	SourceBackoffMaxElapsedTime time.Duration

	// Max time between failed retries to save data.
	SourceBackoffMaxInterval time.Duration

	// How many blocks are downloaded in parallel
	SourceBatchSize int

	// Num of Interactions that are stored in the Store
	// before being inserted into the database in one db transaction and batch.
	StoreBatchSize int

	// After this time all Interactions in Store will be inserted to the database.
	// This is to avoid keeping them in the service for too long when waiting to fill the batch.
	StoreMaxTimeInQueue time.Duration

	// Max time between failed retries to save data.
	StoreMaxBackoffInterval time.Duration
}

func setRelayerDefaults() {
	viper.SetDefault("Relayer.Environment", "dev")
	viper.SetDefault("Relayer.SequencerUrl", "tcp://127.0.0.1:26657")
	viper.SetDefault("Relayer.ArweaveBlockDownloadTimeout", "45s")
	viper.SetDefault("Relayer.ArweaveBlockDownloadMaxElapsedTime", "0s")
	viper.SetDefault("Relayer.ArweaveBlockDownloadMaxInterval", "5s")
	viper.SetDefault("Relayer.SequencerQueueSize", "5")
	viper.SetDefault("Relayer.SourceMaxWorkers", "50")
	viper.SetDefault("Relayer.SourceMaxQueueSize", "1")
	viper.SetDefault("Relayer.SourceBackoffMaxElapsedTime", "30s")
	viper.SetDefault("Relayer.SourceBackoffMaxInterval", "2s")
	viper.SetDefault("Relayer.SourceBatchSize", "10")
	viper.SetDefault("Relayer.StoreBatchSize", "100")
	viper.SetDefault("Relayer.StoreMaxTimeInQueue", "10s")
	viper.SetDefault("Relayer.StoreMaxBackoffInterval", "10s")
}
