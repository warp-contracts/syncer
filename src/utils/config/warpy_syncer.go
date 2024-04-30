package config

import (
	"time"

	"github.com/spf13/viper"
	"github.com/warp-contracts/syncer/src/utils/eth"
)

type WarpySyncer struct {
	// Maximum length of the channel output
	BlockDownloaderChannelSize int

	// How often poll for new block
	BlockDownloaderInterval time.Duration

	// Max worker pool's queue size
	BlockDownloaderMaxQueueSize int

	// Max batch size for the number of blocks downloaded in one iteration
	BlockDownloaderBatchSize int

	// Max time between failed retries to download block
	BlockDownloaderBackoffInterval time.Duration

	// Time between poller task is called from block downloader
	BlockDownloaderPollerInterval int64

	// Block time
	BlockDownloaderBlockTime float64

	// If should download block by header number (for integrations which cannot decode transactions inside the block)
	BlockDownloaderByHeader bool

	// Warpy contract id
	SyncerContractId string

	// Name service contract id, usually the same as SyncerContractId
	SyncerNameServiceContractId string

	// Dre-Warpy URL
	SyncerDreUrl string

	// Warpy API url
	SyncerWarpyApiUrl string

	// Chain to by synced
	SyncerChain eth.Chain

	// Protocol to be synced
	SyncerProtocol eth.Protocol

	// API key
	SyncerApiKey string

	// Warpy admin id
	SyncerInteractionAdminId string

	// Signer for the Warpy interactions
	SyncerSigner string

	// Max time between failed retries to sync transaction
	SyncerDeltaBackoffInterval time.Duration

	// Number of workers that sync transactions
	SyncerDeltaNumWorkers int

	// Max number of transactions that wait in the worker queue
	SyncerDeltaWorkerQueueSize int

	// Data to be searched in transactions
	SyncerDeltaRedstoneData string

	// Number of points assigned in the Warpy interaction
	SyncerDeltaInteractionPoints int64

	// Sommelier contract id to be synced
	SyncerDepositContractId string

	// Sommelier functions to be synced
	SyncerDepositFunctions []string

	// Max time between failed retries to sync transaction
	SyncerDepositBackoffInterval time.Duration

	// Number of workers that sync transactions
	SyncerDepositNumWorkers int

	// Max number of transactions that wait in the worker queue
	SyncerDepositWorkerQueueSize int

	// Max batch size before last block synced will be inserted into database
	StoreBatchSize int

	// After this time last block synced will be inserted into database
	StoreInterval time.Duration

	// Sommelier functions for withdrawal
	StoreDepositWithdrawFunctions []string

	// Name of the deposit assets input name
	StoreDepositDepositAssetsName string

	// Name of the withdraw assets input name
	StoreDepositWithdrawAssetsName string

	// Max time between failed retries to save last block synced
	StoreMaxBackoffInterval time.Duration

	// Maximum length of the channel buffer
	PollerDepositChannelBufferLength int

	// How often to poll the database
	PollerDepositInterval time.Duration

	// How long does it wait for the query response
	PollerDepositTimeout time.Duration

	// Base for the points multiplication
	PollerDepositPointsBase int64

	// How much time should pass until we include transaction in rewards (in seconds)
	PollerDepositSecondsForSelect int64

	// Max time between failed retries to write interaction
	WriterBackoffInterval time.Duration

	// Timeout for HTTP requests
	WriterHttpRequestTimeout time.Duration

	// Writer splits interaction into chunks with max size of
	WriterInteractionChunkSize int
}

func setWarpySyncerDefaults() {
	viper.SetDefault("WarpySyncer.BlockDownloaderInterval", "30s")
	viper.SetDefault("WarpySyncer.BlockDownloaderMaxQueueSize", 1000)
	viper.SetDefault("WarpySyncer.BlockDownloaderBatchSize", 100)
	viper.SetDefault("WarpySyncer.BlockDownloaderBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.BlockDownloaderChannelSize", 100)
	viper.SetDefault("WarpySyncer.BlockDownloaderPollerInterval", 3600) // should be 1h: 60 * 60 seconds
	viper.SetDefault("WarpySyncer.BlockDownloaderBlockTime", float64(0.26))
	viper.SetDefault("WarpySyncer.BlockDownloaderByHeader", true)
	viper.SetDefault("WarpySyncer.SyncerContractId", "mdxBOJ3cy98lqJoPZf7EW0iU4jaqePC3XZRkzoWU1QY")
	viper.SetDefault("WarpySyncer.SyncerNameServiceContractId", "p5OI99-BaY4QbZts266T7EDwofZqs-wVuYJmMCS0SUU")
	viper.SetDefault("WarpySyncer.SyncerChain", eth.Manta)
	viper.SetDefault("WarpySyncer.SyncerProtocol", eth.LayerBank)
	viper.SetDefault("WarpySyncer.SyncerDreUrl", "https://dre-warpy.warp.cc")
	viper.SetDefault("WarpySyncer.SyncerWarpyApiUrl", "https://api-warpy.warp.cc")
	viper.SetDefault("WarpySyncer.SyncerApiKey", "")
	viper.SetDefault("WarpySyncer.SyncerInteractionAdminId", "769844280767807520")
	viper.SetDefault("WarpySyncer.SyncerSigner", "")
	viper.SetDefault("WarpySyncer.SyncerDeltaInteractionPoints", 20)
	viper.SetDefault("WarpySyncer.SyncerDeltaBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.SyncerDeltaNumWorkers", "50")
	viper.SetDefault("WarpySyncer.SyncerDeltaWorkerQueueSize", "10")
	viper.SetDefault("WarpySyncer.SyncerDeltaRedstoneData", "000002ed57011e0000")
	viper.SetDefault("WarpySyncer.SyncerDeltaNumWorkers", "50")
	viper.SetDefault("WarpySyncer.SyncerDeltaWorkerQueueSize", "10")
	viper.SetDefault("WarpySyncer.SyncerDepositContractId", "0xB7A23Fc0b066051dE58B922dC1a08f33DF748bbf")
	viper.SetDefault("WarpySyncer.SyncerDepositBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.SyncerDepositFunctions", []string{"supply", "redeemToken"})
	viper.SetDefault("WarpySyncer.StoreDepositWithdrawFunctions", []string{"redeemToken"})
	viper.SetDefault("WarpySyncer.StoreBatchSize", "500")
	viper.SetDefault("WarpySyncer.StoreInterval", "2s")
	viper.SetDefault("WarpySyncer.StoreMaxBackoffInterval", "30s")
	viper.SetDefault("WarpySyncer.StoreDepositDepositAssetsName", "lAmount")
	viper.SetDefault("WarpySyncer.StoreDepositWithdrawAssetsName", "uAmount")
	viper.SetDefault("WarpySyncer.PollerDepositChannelBufferLength", 100)
	viper.SetDefault("WarpySyncer.PollerDepositInterval", "1m")
	viper.SetDefault("WarpySyncer.PollerDepositTimeout", "90s")
	viper.SetDefault("WarpySyncer.PollerDepositPointsBase", 1000)
	viper.SetDefault("WarpySyncer.PollerDepositSecondsForSelect", 3600)
	viper.SetDefault("WarpySyncer.WriterBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.WriterHttpRequestTimeout", "30s")
	viper.SetDefault("WarpySyncer.WriterInteractionChunkSize", 50)
}
