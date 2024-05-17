package config

import (
	"time"

	"github.com/spf13/viper"
)

type Forwarder struct {
	// Last block height forwarder will attempt to set last sort keys
	FetcherLastSortKeySettingBlockHeight uint64

	// If true, L1 interactions get last sort key assigned and saved before being sent further
	FetcherLastSortKeySettingEnabled bool

	// How many L1 interactions are fetched from the DB at once
	FetcherBatchSize int

	// Interactions are saved to this Redis channel
	PublisherRedisChannelName string

	// Interactions are saved to this AppSync channel
	PublisherAppSyncChannelName string

	// How long to wait before after receiving a new block height before sending L1 interactions
	// This delay ensures sequencer finishes handling requests in time
	HeightDelay time.Duration

	// How many L1 interactions are cached in queue
	// This should be at least 1000 since this is how many tx are in Arweave block
	ArweaveFetcherQueueSize int

	// How long to wait to send all L1 interactions from a given block
	// There's 2m between blocks, so this should be at most 2m
	ArweaveFetcherBlockSendTimeout time.Duration

	// How many L2 interactions are cached in queue
	// Those are L2 interactions streamed live from the database
	InteractionsStreamerQueueSize int
}

func setForwarderDefaults() {
	viper.SetDefault("Forwarder.FetcherLastSortKeySettingBlockHeight", "0")
	viper.SetDefault("Forwarder.FetcherLastSortKeySettingEnabled", "true")
	viper.SetDefault("Forwarder.FetcherBatchSize", "10")
	viper.SetDefault("Forwarder.PublisherRedisChannelName", "contracts")
	viper.SetDefault("Forwarder.PublisherAppSyncChannelName", "interactions")
	viper.SetDefault("Forwarder.HeightDelay", "1s")
	viper.SetDefault("Forwarder.ArweaveFetcherQueueSize", "3000")
	viper.SetDefault("Forwarder.ArweaveFetcherBlockSendTimeout", "300s")
	viper.SetDefault("Forwarder.InteractionsStreamerQueueSize", "10")
}
