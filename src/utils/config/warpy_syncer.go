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

	// RPC API key
	SyncerRpcApiKey string

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
	SyncerDepositContractIds []string

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
	SyncerDepositTokens []string

	// If assets for withdrawal should be taken from transaction receipt, contract abi for events should be provided
	SyncerDepositLogContractAbi string

	// If assets for withdrawal should be taken from transaction receipt, name of the log should be provided
	SyncerDepositWithdrawLog string

	// If assets for deposit should be taken from transaction receipt, name of the log should be provided
	SyncerDepositDepositLog string

	// Max batch size before last block synced will be inserted into database
	StoreBatchSize int

	// After this time last block synced will be inserted into database
	StoreInterval time.Duration

	// Functions for withdrawal
	StoreDepositWithdrawFunctions []string

	// Names of the deposit assets input name
	AssetsCalculatorDepositAssetsNames []string

	// Names of the withdraw assets input name
	AssetsCalculatorWithdrawAssetsNames []string

	// Names of the deposit assets input name from log
	AssetsCalculatorDepositLogAssetsNames []string

	// Names of the withdraw assets name from log
	AssetsCalculatorWithdrawLogAssetsNames []string

	// Token name if not ETH or wrapped ETH
	AssetsCalculatorToken string

	// Inputs map asset name field
	AssetsCalculatorInputTokenName string

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

	// Maximum points cap rewarded daily
	WriterPointsCap int64

	// How long the integration will last
	WriterIntegrationDurationInSec int64

	// How much time should pass until we include transaction in rewards (in seconds)
	PollerDepositSecondsForSelect int64

	// Max time between failed retries to write interaction
	WriterBackoffInterval time.Duration

	// Timeout for HTTP requests
	WriterHttpRequestTimeout time.Duration

	// Writer splits interaction into chunks with max size of
	WriterInteractionChunkSize int

	// API key for sequencer request
	WriterApiKey string
}

func setWarpySyncerDefaults() {
	viper.SetDefault("WarpySyncer.BlockDownloaderInterval", "10s")
	viper.SetDefault("WarpySyncer.BlockDownloaderMaxQueueSize", 1000)
	viper.SetDefault("WarpySyncer.BlockDownloaderBatchSize", 100)
	viper.SetDefault("WarpySyncer.BlockDownloaderBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.BlockDownloaderChannelSize", 100)
	viper.SetDefault("WarpySyncer.BlockDownloaderPollerInterval", 3600) // should be 1h: 60 * 60 seconds
	viper.SetDefault("WarpySyncer.BlockDownloaderBlockTime", float64(2))
	viper.SetDefault("WarpySyncer.BlockDownloaderByHeader", false)
	viper.SetDefault("WarpySyncer.SyncerContractId", "mdxBOJ3cy98lqJoPZf7EW0iU4jaqePC3XZRkzoWU1QY")
	viper.SetDefault("WarpySyncer.SyncerNameServiceCFproxyontractId", "mdxBOJ3cy98lqJoPZf7EW0iU4jaqePC3XZRkzoWU1QY")
	viper.SetDefault("WarpySyncer.SyncerChain", eth.Base)
	viper.SetDefault("WarpySyncer.SyncerProtocol", eth.ZeroLend)
	viper.SetDefault("WarpySyncer.SyncerDreUrl", "https://dre-warpy.warp.cc")
	viper.SetDefault("WarpySyncer.SyncerWarpyApiUrl", "https://api-warpy.warp.cc")
	viper.SetDefault("WarpySyncer.SyncerApiKey", "")
	viper.SetDefault("WarpySyncer.SyncerRpcApiKey", "")
	viper.SetDefault("WarpySyncer.SyncerInteractionAdminId", "769844280767807520")
	viper.SetDefault("WarpySyncer.SyncerSigner", "")
	viper.SetDefault("WarpySyncer.SyncerDeltaInteractionPoints", 20)
	viper.SetDefault("WarpySyncer.SyncerDeltaBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.SyncerDeltaNumWorkers", "50")
	viper.SetDefault("WarpySyncer.SyncerDeltaWorkerQueueSize", "10")
	viper.SetDefault("WarpySyncer.SyncerDeltaRedstoneData", "000002ed57011e0000")
	viper.SetDefault("WarpySyncer.SyncerDeltaNumWorkers", "50")
	viper.SetDefault("WarpySyncer.SyncerDeltaWorkerQueueSize", "10")
	viper.SetDefault("WarpySyncer.SyncerDepositContractIds", []string{"0x766f21277087E18967c1b10bF602d8Fe56d0c671"})
	viper.SetDefault("WarpySyncer.SyncerDepositBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.SyncerDepositFunctions", []string{"supply", "withdraw"})
	viper.SetDefault("WarpySyncer.SyncerDepositMarkets", []string{
		// wETH
		"0x952083cde7aaa11AB8449057F7de23A970AA8472",
		"0xf9F9779d8fF604732EBA9AD345E6A27EF5c2a9d6",
		// rsETH
		"0x6Ae79089b2CF4be441480801bb741A531d94312b",
		"0xED99fC8bdB8E9e7B8240f62f69609a125A0Fbf14",
		// ezETH
		"0x5E03C94Fc5Fb2E21882000A96Df0b63d2c4312e2",
		"0x35f3dB08a6e9cB4391348b0B404F493E7ae264c0",
	})
	viper.SetDefault("WarpySyncer.SyncerDepositTokens", []string{"0xecac9c5f704e954931349da37f60e39f515c11c1"})
	viper.SetDefault("WarpySyncer.SyncerDepositLogContractAbi", "0x766f21277087E18967c1b10bF602d8Fe56d0c671")
	viper.SetDefault("WarpySyncer.SyncerDepositWithdrawLog", "Withdraw")
	viper.SetDefault("WarpySyncer.SyncerDepositDepositLog", "Supply")
	viper.SetDefault("WarpySyncer.StoreDepositWithdrawFunctions", []string{"withdraw"})
	viper.SetDefault("WarpySyncer.StoreBatchSize", "500")
	viper.SetDefault("WarpySyncer.StoreInterval", "2s")
	viper.SetDefault("WarpySyncer.StoreMaxBackoffInterval", "30s")
	viper.SetDefault("WarpySyncer.AssetsCalculatorDepositAssetsNames", []string{"amount"})
	viper.SetDefault("WarpySyncer.AssetsCalculatorDepositLogAssetsNames", []string{"amount"})
	viper.SetDefault("WarpySyncer.AssetsCalculatorWithdrawAssetsNames", []string{})
	viper.SetDefault("WarpySyncer.AssetsCalculatorWithdrawLogAssetsNames", []string{"amount"})
	viper.SetDefault("WarpySyncer.AssetsCalculatorInputTokenName", "asset")
	viper.SetDefault("WarpySyncer.PollerDepositChannelBufferLength", 100)
	viper.SetDefault("WarpySyncer.PollerDepositInterval", "1m")
	viper.SetDefault("WarpySyncer.PollerDepositTimeout", "90s")
	viper.SetDefault("WarpySyncer.PollerDepositPointsBase", 1000)
	viper.SetDefault("WarpySyncer.WriterPointsCap", 2000000)
	viper.SetDefault("WarpySyncer.WriterIntegrationDurationInSec", 432000)
	viper.SetDefault("WarpySyncer.PollerDepositSecondsForSelect", 3600)
	viper.SetDefault("WarpySyncer.WriterBackoffInterval", "3s")
	viper.SetDefault("WarpySyncer.WriterHttpRequestTimeout", "30s")
	viper.SetDefault("WarpySyncer.WriterInteractionChunkSize", 50)
	viper.SetDefault("WarpySyncer.WriterApiKey", "")
}
