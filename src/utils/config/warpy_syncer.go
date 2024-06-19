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

	// Accepted markets in which token is being deposited
	SyncerDepositMarkets []string

	// Supported token
	SyncerDepositToken string

	// Max batch size before last block synced will be inserted into database
	StoreBatchSize int

	// After this time last block synced will be inserted into database
	StoreInterval time.Duration

	// Functions for withdrawal
	StoreDepositWithdrawFunctions []string

	// Names of the deposit assets input name
	StoreDepositDepositAssetsNames []string

	// Names of the withdraw assets input name
	StoreDepositWithdrawAssetsNames []string

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
	viper.SetDefault("WarpySyncer.SyncerContractId", "p5OI99-BaY4QbZts266T7EDwofZqs-wVuYJmMCS0SUU")
	viper.SetDefault("WarpySyncer.SyncerNameServiceContractId", "p5OI99-BaY4QbZts266T7EDwofZqs-wVuYJmMCS0SUU")
	viper.SetDefault("WarpySyncer.SyncerChain", eth.Arbitrum)
	viper.SetDefault("WarpySyncer.SyncerProtocol", eth.Pendle)
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
	viper.SetDefault("WarpySyncer.SyncerDepositContractId", "0x888888888889758f76e7103c6cbf23abbf58f946")
	viper.SetDefault("WarpySyncer.SyncerDepositBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.SyncerDepositFunctions", []string{"swapExactTokenForPt", "swapExactPtForToken"})
	viper.SetDefault("WarpySyncer.SyncerDepositMarkets", []string{
		// wETH
		"0x952083cde7aaa11AB8449057F7de23A970AA8472",
		// rsETH
		"0x6Ae79089b2CF4be441480801bb741A531d94312b",
		// ezETH
		"0x5E03C94Fc5Fb2E21882000A96Df0b63d2c4312e2",
	})
	viper.SetDefault("WarpySyncer.SyncerDepositToken", "0x6A0d9584D88D22BcaD7D4F83E7d6AB7949895DDF")
	viper.SetDefault("WarpySyncer.StoreDepositWithdrawFunctions", []string{"swapExactPtForToken"})
	viper.SetDefault("WarpySyncer.StoreBatchSize", "500")
	viper.SetDefault("WarpySyncer.StoreInterval", "2s")
	viper.SetDefault("WarpySyncer.StoreMaxBackoffInterval", "30s")
	viper.SetDefault("WarpySyncer.StoreDepositDepositAssetsNames", []string{"input", "netTokenIn"})
	viper.SetDefault("WarpySyncer.StoreDepositWithdrawAssetsNames", []string{"exactPtIn"})
	viper.SetDefault("WarpySyncer.PollerDepositChannelBufferLength", 100)
	viper.SetDefault("WarpySyncer.PollerDepositInterval", "1m")
	viper.SetDefault("WarpySyncer.PollerDepositTimeout", "90s")
	viper.SetDefault("WarpySyncer.PollerDepositPointsBase", 1000)
	viper.SetDefault("WarpySyncer.PollerDepositSecondsForSelect", 3600)
	viper.SetDefault("WarpySyncer.WriterBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.WriterHttpRequestTimeout", "30s")
	viper.SetDefault("WarpySyncer.WriterInteractionChunkSize", 50)
}
