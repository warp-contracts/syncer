package config

import (
	"time"

	"github.com/spf13/viper"
)

type RedstoneTxSyncer struct {
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

	// Max time between failed retries to sync transaction
	SyncerBackoffInterval time.Duration

	// Number of workers that sync transactions
	SyncerNumWorkers int

	// Max number of transactions that wait in the worker queue
	SyncerWorkerQueueSize int

	// Timeout for HTTP requests
	SyncerHttpRequestTimeout time.Duration

	// Data to be searched in transactions
	SyncerRedstoneData string

	// Warpy contract id
	SyncerContractId string

	// Name service contract id, usually the same as SyncerContractId
	SyncerNameServiceContractId string

	// Dre-Warpy URL
	SyncerDreUrl string

	// Warpy API url
	SyncerWarpyApiUrl string

	// Number of points assigned in the Warpy interaction
	SyncerInteractionPoints int

	// Admin id passed in the Warpy interaction
	SyncerInteractionAdminId string

	// Arweave Signer JWK of the interactions sent to Warpy
	SyncerSigner string

	// Max batch size before last block synced will be inserted into database
	StoreBatchSize int

	// After this time last block synced will be inserted into database
	StoreInterval time.Duration

	// Max time between failed retries to save last block synced
	StoreMaxBackoffInterval time.Duration
}

func setRedstoneTxSyncerDefaults() {
	viper.SetDefault("RedstoneTxSyncer.BlockDownloaderInterval", "10s")
	viper.SetDefault("RedstoneTxSyncer.BlockDownloaderMaxQueueSize", 1000)
	viper.SetDefault("RedstoneTxSyncer.BlockDownloaderBatchSize", 100)
	viper.SetDefault("RedstoneTxSyncer.BlockDownloaderBackoffInterval", "3s")
	viper.SetDefault("RedstoneTxSyncer.SyncerBackoffInterval", "3s")
	viper.SetDefault("RedstoneTxSyncer.BlockDownloaderChannelSize", 100)
	viper.SetDefault("RedstoneTxSyncer.SyncerNumWorkers", "50")
	viper.SetDefault("RedstoneTxSyncer.SyncerWorkerQueueSize", "10")
	viper.SetDefault("RedstoneTxSyncer.SyncerHttpRequestTimeout", "30s")
	viper.SetDefault("RedstoneTxSyncer.SyncerRedstoneData", "000002ed57011e0000")
	viper.SetDefault("RedstoneTxSyncer.SyncerContractId", "RfUarG0sNgwREiKXOO1YeWK6-f3MGb6u4XqMccQwBRc")
	viper.SetDefault("RedstoneTxSyncer.SyncerNameServiceContractId", "p5OI99-BaY4QbZts266T7EDwofZqs-wVuYJmMCS0SUU")
	viper.SetDefault("RedstoneTxSyncer.SyncerDreUrl", "https://dre-warpy.warp.cc")
	viper.SetDefault("RedstoneTxSyncer.SyncerWarpyApiUrl", "https://api-warpy.warp.cc")
	viper.SetDefault("RedstoneTxSyncer.SyncerInteractionPoints", 20)
	viper.SetDefault("RedstoneTxSyncer.SyncerInteractionAdminId", "769844280767807520")
	viper.SetDefault("RedstoneTxSyncer.SyncerSigner", `{
		"d": "IVv3IzUPbj2yJP9qqJcH3cVI86jWdhZCpNoomLeJaH0rpKnujzlDSADC2yuFNBnS_sIthk1-w83_bkTwwOOCAn_9LZbkKYEd2onZ7iWAh--tMB5ijNHv0acn64TZjS-5aH6WgfsxwCjrXj57ejnh7GaterucVpTX_RlGtpp5IWY5ISM-5JLBm2wLLnXjhsJD51a03eClxy0MAclG6suOkm2pRF7yl1sJjQ23kZ7xExpO-Lb_j8o1JEGao5xI1TPWdJyovuhPrWK14l3JXU9URz6IKFH9xuvbWjqWhyVQVjUBBWg5B5DbzQhI_6tPVHb8eUBP9L9BNkRyr5cWU1SCYynzEa9_1cXjLuYNtTUB9358bkveYiZRlvSjCYoNd6lSFtESbyMfvmU2FF7gnduVqzdTPuisfHHNYQKCall-emCt9Oiy26OJ2uMX-dfqutcZd65OlJN5KG65h6D8cp7xjDlwHx4VeK2qI-dyzOS6ufZlG0nrNEfzRDekmRsFCgZxJUjc0JjCMde5LRKZhsmltntizeaURw69dnNTrtrLFQLlo6X3wEHzyjFNqaqJDQmB6UnpdOjZp6FeotV02FpeqhJZ8pA1kYywO9LFB-iciy7h-bufHoK5Owti-CwOMADdwzYPPaKrbhc7ZhAuogQTMfFSHJtL5_le_Y-k8FTtu4E",
		"dp": "phZwSYPUvAO-231R-_IuLMHB294qzoiHeg1GUBEAvf4PqA95dgQAXUQUTEVUBuOvJ89g4Zubz3QcRabzEeySGDHLhF0x5BdUCmZugiQJ_MphBPTa82PPDWWohPTdztt8L-2mXWAJRHQqesT4zix7cKYao9wbWvG-9i0sDzk9hfFT9HNM8yr5-Sp089so-5jro-48ZWa97nhsOKDvNamHX9BdOX-TSl97txlSf5IjgXeGUImgIcIgZAdnp7cWjo2rYodyaeJ_yh_dGEnVL1XauVJ5gochLIKcIIZWaO0ENqvPJdly_TT7FUHG-uLUicSGRJuloBooZzLUzMuasSZwoQ",
		"dq": "X_oppBgiMcI6fyuvlTI9YaveiJmLWI_B2T1IsdU0xPS1PvPdjLq5ArK7NpqlkWsaF3Y4eR96uPniNPGrnvl7Z4A383G7zOXtlFzuYZxvXMGs9G46VNVXxT0vvO9Htm4Zp8W11eW9MneKXdeJ-uMUcTw3vlCgXG8x9C2CcTqRN_J3PNiWmkHT2FE5Tbqwj36MPPOOInI-22k3UG2OX2qOrQoFD6SPgRoRLJmRLDl_ktJ1rQus187FfNgmB77-qeg_p772jwLxnzIvay4WmehJdI1wdp_JlKmQkEqknAq_ab0ltLcofqCR4-_2MkFMLksqVDilUtQkH3Od0QYIlbM9kw",
		"e": "AQAB",
		"ext": true,
		"kty": "RSA",
		"n": "xEDoW3dIO93QcmK3G1bgNrguKoI1eSsgtBd5IERwJOtpqM2cBDlqkMbMhcy3dzL-0YPSPAB78HudvhnmNlTRWas9zqPX7nj0CtcDlbntAWIyjUXUUbqdRHUkvOpUzEcdU-x9ZLFPOJfAMAZ5Wh0kdASjptyWzQLRErBkX_4nzIJm79SdLkYvkr5toJxPtdxlVXRgcEU1ZuythSGRPKH_CNRsJVMqJxqWBGU4JgVks1LeVZ-sUvQSWVGCMCRRqPdaAEFjFLTeNknLuMDvngc00mE9GeESISENSNiVUc5Zy7pOX0I9NuuUOFl8XjnjIbJBoxX_MnJNhj4pFu3X-l20_ejlKlYrkSFeWHcw0u2_wsCrGuwsNQrrL1iUHSe7ohhB7HLmJ-DQd1BaatUMsRTxLpGR1n_fgq_3xbtm0xsZ83dLJkr8ewNtp63v18LBzJIJmaYW1rICBnmEK8IChDIWjZOk5tQ7ghMNO10bgrnI0Ba0l_arZM3lPISv74kRG_BuS3MiDUqZ5bYD_S5QYknWf6LzBWlSd0aOVScA1ZFBtnuLu4DETCDNivAXqGYbsvDHJsytXgeVWiRog44E1hHR2Xd2W2ax5KsZaxRGwl4KxUF-WnMu8kVgPZFUkIUPQpy7nQNFkyb-F6wemYRZeaPkKy96HD3Zfy_yvEVH4r_LJZs",
		"p": "-2r7Ncw3A6IqNvgGrWtPmGcdljQlNYhtGXFCyj8Juhm-Tn8jyGb45mYpy6rOcCIwiAn8PsCVvJ1DGZlUdJp5DoKPA6KEGviDzO0ANFV0z71h4X_sLk3CZJ7uQ7NuLqxrToZDf2q_ENA6Xg_MFAqC2dKVYCCKdGAiS5flZMEf_B0-0aw1WbNfnXGUKNMNyzIgXH3I10EBFVYfNBnTySGUmmZ3twmeimfYfgyFf56SKyLNj91IUCWqxSPj8XhYHUJYGxMs-4wE8m7ysk7RZnGpQyro-wBXWHhMjqM3wXvWiSjSm_1zVQqcGCdt_6fqaLb5Uy82FFDkxcB4VyMh4uQKsQ",
		"q": "x9SNAr0sk186_9z8WwGGis5_HxOXfiiiqqNO_OaKbHTW1iYdbgQpdPlF-nft8gh4dAKzGQ6hPz0H64lcjL22LWUYjPDkGeByubHuFFbFGlnZpWBXNbceHvYxBrfLBRC2vug1QE21-c8Hww0VnNX0macM0E2sxruEDJXcvdz3jdf-42lPCNPlX73HVmmJACWzubKEsl_VK1MdwWZb_cNL7w6AdwOcug-_YZfMlPv9I8sTMqNwNKppWcrqV1bz0Or04ds1ifA-WR52eaodU8jSMa7j92GShKxtjJ6yaMutLaNtMxsuk1QTAKyAGGUH3HhW_BiS8P2LIGhW5binojWwCw",
		"qi": "XqpyET1rXxpqflIE_5fpVYzpJy316JgBcoFoaQwJXBV2S-AkiOgSHVP_OClZXj2ondHHpShvNbSmFZ8NDunbZhNqDWpXYWFJsdq8-Hcid-c0kipCfh75i799EdLs2HS8zAbbJiVhl5I0QeTE0n3mEUsNWDSMC0pIbZtKuc1Ij849rIxIDhMOKjEMCNUQJVn-FcajTttoamnUHzb4whFmgnMm8JWVDwdFK0Yt4TbchrHg4gpmGHzn1LD4mUPeqstd_JKgZQYMzZawAupN9C3SXDCYjAI6Glskjm-M5eC3yTEFnOE74cHymtI61rU-4-n2aPzMMPsJsLm7U8hzKkHEZg"
	}`)
	viper.SetDefault("RedstoneTxSyncer.StoreBatchSize", "500")
	viper.SetDefault("RedstoneTxSyncer.StoreInterval", "2s")
	viper.SetDefault("Syncer.StoreMaxBackoffInterval", "30s")
}
