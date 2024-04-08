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

	// API key for Arbiscan
	SyncerArbiscanApiKey string

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
	SyncerSommelierContractId string

	// Sommelier functions to be synced
	SyncerSommelierFunctions []string

	// Max time between failed retries to sync transaction
	SyncerSommelierBackoffInterval time.Duration

	// Number of workers that sync transactions
	SyncerSommelierNumWorkers int

	// Max number of transactions that wait in the worker queue
	SyncerSommelierWorkerQueueSize int

	// Max batch size before last block synced will be inserted into database
	StoreBatchSize int

	// After this time last block synced will be inserted into database
	StoreInterval time.Duration

	// Sommelier functions for withdrawal
	StoreSommelierWithdrawFunctions []string

	// Max time between failed retries to save last block synced
	StoreMaxBackoffInterval time.Duration

	// Maximum length of the channel buffer
	PollerSommelierChannelBufferLength int

	// How often to poll the database
	PollerSommelierInterval time.Duration

	// How long does it wait for the query response
	PollerSommelierTimeout time.Duration

	// Cron which indicates how often to poll for new records
	PollerSommelierCron string

	// Base for the points multiplication
	PollerSommelierPointsBase int64

	// How much time should pass until we include transaction in rewards (in seconds)
	PollerSommelierSecondsForSelect int64

	// Max time between failed retries to write interaction
	WriterBackoffInterval time.Duration

	// Timeout for HTTP requests
	WriterHttpRequestTimeout time.Duration
}

func setWarpySyncerDefaults() {
	viper.SetDefault("WarpySyncer.BlockDownloaderInterval", "10s")
	viper.SetDefault("WarpySyncer.BlockDownloaderMaxQueueSize", 1000)
	viper.SetDefault("WarpySyncer.BlockDownloaderBatchSize", 100)
	viper.SetDefault("WarpySyncer.BlockDownloaderBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.BlockDownloaderChannelSize", 100)
	viper.SetDefault("WarpySyncer.SyncerContractId", "p5OI99-BaY4QbZts266T7EDwofZqs-wVuYJmMCS0SUU")
	viper.SetDefault("WarpySyncer.SyncerNameServiceContractId", "p5OI99-BaY4QbZts266T7EDwofZqs-wVuYJmMCS0SUU")
	viper.SetDefault("WarpySyncer.SyncerChain", eth.Arbitrum)
	viper.SetDefault("WarpySyncer.SyncerProtocol", eth.Sommelier)
	viper.SetDefault("WarpySyncer.SyncerDreUrl", "https://dre-warpy.warp.cc")
	viper.SetDefault("WarpySyncer.SyncerWarpyApiUrl", "https://api-warpy.warp.cc")
	viper.SetDefault("WarpySyncer.SyncerArbiscanApiKey", "")
	viper.SetDefault("WarpySyncer.SyncerInteractionAdminId", "769844280767807520")
	viper.SetDefault("WarpySyncer.SyncerSigner", "")
	viper.SetDefault("WarpySyncer.SyncerDeltaInteractionPoints", 20)
	viper.SetDefault("WarpySyncer.SyncerDeltaBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.SyncerDeltaNumWorkers", "50")
	viper.SetDefault("WarpySyncer.SyncerDeltaWorkerQueueSize", "10")
	viper.SetDefault("WarpySyncer.SyncerDeltaRedstoneData", "000002ed57011e0000")
	viper.SetDefault("WarpySyncer.SyncerDeltaNumWorkers", "50")
	viper.SetDefault("WarpySyncer.SyncerDeltaWorkerQueueSize", "10")
	viper.SetDefault("WarpySyncer.SyncerSommelierContractId", "0xC47bB288178Ea40bF520a91826a3DEE9e0DbFA4C")
	viper.SetDefault("WarpySyncer.SyncerSommelierBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.SyncerSommelierFunctions", []string{"deposit", "multiAssetDeposit", "redeem"})
	viper.SetDefault("WarpySyncer.StoreSommelierWithdrawFunctions", []string{"redeem"})
	viper.SetDefault("WarpySyncer.StoreBatchSize", "500")
	viper.SetDefault("WarpySyncer.StoreInterval", "2s")
	viper.SetDefault("WarpySyncer.StoreMaxBackoffInterval", "30s")
	viper.SetDefault("WarpySyncer.PollerSommelierChannelBufferLength", 100)
	viper.SetDefault("WarpySyncer.PollerSommelierInterval", "1m")
	viper.SetDefault("WarpySyncer.PollerSommelierTimeout", "90s")
	viper.SetDefault("WarpySyncer.PollerSommelierCron", "0 * * * * *")
	viper.SetDefault("WarpySyncer.PollerSommelierPointsBase", 1000)
	viper.SetDefault("WarpySyncer.PollerSommelierSecondsForSelect", 3600)
	viper.SetDefault("WarpySyncer.WriterBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.WriterHttpRequestTimeout", "30s")
}
