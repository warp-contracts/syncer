package config

import (
	"bytes"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config stores global configuration
type Config struct {
	// SQL Database
	DBPort        string
	DBHost        string
	DBUser        string
	DBPassword    string
	DBName        string
	DBSSLMode     string
	DBPingTimeout time.Duration

	// REST API
	RESTListenAddress string

	// Logging
	LogLevel string
	LogPath  string

	ArNodeUrl               string
	ArConcurrentConnections int
	ArStableDistance        int64
	ArWalletJwk             string
	ArRequestTimeout        time.Duration

	// How often should Arweave network info be downloaded
	PollerNetworkInfoTimeout time.Duration

	// How many parallel messages are exchanged with Arweave node. HTTP/2 is used so it's still one TCP connection per node.
	PollerWorkerPoolSize int

	// Received Arweave transactions converted to interactions and temporarily stored in the channel
	// Normally interactions are passed to the Store right away, but if Store is in the middle of transaction it's not receiving data.
	// So this capacity should account for interactions that may appear during a few second window when the previous batch is inserted to the database.
	ListenerQueueSize int

	// Num of Interactions that are stored in the Store
	// before being inserted into the database in one db transaction and batch.
	StoreBatchSize int

	// After this time all Interactions in Store will be inserted to the database.
	// This is to avoid keeping them in the service for too long when waiting to fill the batch.
	StoreMaxTimeInQueue time.Duration
}

func setDefaults() {
	viper.SetDefault("DBPort", "7654")
	viper.SetDefault("DBHost", "127.0.0.1")
	viper.SetDefault("DBUser", "postgres")
	viper.SetDefault("DBPassword", "postgres")
	viper.SetDefault("DBName", "warp")
	viper.SetDefault("DBSSLMode", "disable")
	viper.SetDefault("DBPingTimeout", "15s")
	viper.SetDefault("RESTListenAddress", ":3333")
	viper.SetDefault("LogLevel", "DEBUG")
	viper.SetDefault("LogPath", "")
	viper.SetDefault("ArNodeUrl", "https://arweave.net")
	viper.SetDefault("ArConcurrentConnections", "50")
	viper.SetDefault("ArStableDistance", "15")
	viper.SetDefault("ArWalletJwk", `{"d":"hJj6SgBUGauzyo2-lCuYHuMBsYbC0Wh4qo5q1GOET8TRzexA3E_latv84xdVx7v_2MgWwE4ZchigU0dt5t_Y4G5kchiAdAim0Wlyb_mNFk2LtRknE6FEJ__qKZKJUlbMaWl6LaOp5RAxdXNakF1aRu-UUIeKHy-CS3kNiZgsM1AauI5DI7a7hwn5SiEGTYj-OLTrlc6PPW9LOM4sBcTlHIc67OJaxVMBYOqes9_4Q7jEdVXa8dc_quXUtrAppntTInzi9OH54CNUGTFRxIDhTdfowgVNPWsE5P-minN5hiItyrbxm62hr7_JK9exuFdr2sNZxKc256aRsDGODUK_N0zvmztRdVHUEFzH6Afkbze4_pdKQ7SOFWtgNaOboh1ZY671KDfUu3ZAanLVCdQIl1AqrAu1beB7VCJXlTdPaRCVeDi1_jE_VzRXRTghlSgLzqU-RHvYi6p57WlZxLQ80QZunC3vfNo8zg0YB-Uoblp7WE9xWdfVcOE_ciaVQauhTgJHlw6_Q5GpIx2K1FjFHAXfx9fKbUROTvyP2lzzlm9nl-Tn8RP82qzQKwGd-3e9yPE5-7Ur1WEbCuGL_L86i8oHwWjT_1vfDyUl0Pyd1dkoTzwapSOm2Qkvi7CZjm9CUU5eZh7bdgXtixs6waTYWz-ZFA1mNCTiM7LP3d4kVaE","dp":"IlabSoMsFZOccPjh3BtcXu8Aax8AQPQcgbkx_YGAWBHDPsh2Q0Yw5TprVn44LBIfa71fM1599i-SU_H9kPqgTeOknrYaoIeRohMtL7NV6cdWa_Vgz7GQHA1xkeBcW9WT-6w6t3-Rhy3XCXapwceFcPW8zRVad0Epttl8ULqJqnLwN57FrxDSnCF00BdpWhuA4H4RJkAmfn-Z5k4TAHm4Xdn-gmogAOVWvAVLIO4xicC4xkGHBZ-ZlMVLix8l829XT6usM9rJWJq3P85bZvkNvzZSDOjN6zXaNHOaXZe-GlErfcVtEgE3wU4q_NDS9qbfHrvc4QCL-Y_9r1g0i3lfsQ","dq":"N1J2RHCMioMgZGxxTF12wHONYmq7rh-dDGFD8tiBHStnXHPPyMk8PoJaV7Qr21QXg9rV4FV75cKo5Vkht7-jXQliB7Iqoa5tXdX3IT5d7noNePd_G3vOR7eVhdU3vVKpdeywXwIaFVJcLFp9K7orhw1jmQw4dT9FX4n4jDRG1qb2J7xpUdlo17axcinNdjRoOl1q3zBqKJoWytUi55S2doI9E15mNUqga6wMJK9JHZOUAery8q_TJ_KVmDLe3xMfT1gK8THgXm1iJm3TGo8Lhdl55VygbvfAnTij9IyX61qxpSXNa7m5BBf1jIPm77sUh61YJVnCF3qL_jahGZb6IQ","e":"AQAB","ext":true,"kty":"RSA","n":"6oyWpkM7Hk3W8e3LTpOzg6Yt8EblCjwu1xpOtKkKdMedZ8hF0X1rvUpxLC5wO5m8PHjjBiC1TwvN8kMWHN3S0DHIXg9NNhwSllOh7dBb3mj05NLan-Pc2lDNQKMWDDB4D_XamWfK4lg9LMskTQ4ZmFdqM3YoiV-uJ-e9k2SoSUV0kbdPINxwJBQRVHcDfH3yOGBSU2ZPfN-nZGfl78hbN2AxAOg2_4A_Jy1ksJmIKHg6W-nWA-mDSHbXlSDu85xnE2qDp1CnZG6jSnQhCldZf3ZoIH1AdSINdcHBW8Jk3QWKfO4pZId3AaCEFLwW8Kegt9g7bCRJO9VV7s3BVfJnv45KZ4FCC4jscYWsHWRTRSRJ2NrAXrlN8ScbjlaALPdpQFszMjhPfQZYWPhy8V0iWgwZfF4qpBFkO35FVvnp_nvJmGPjpWJVkyZESlka4zirPC8Tn6uhfhI6Rnk6Z6H9bJ-XzISxL1KUTWRQf51JoiQoHu6LtN9P0tVwPhT0Ls9xI4Zh9veAQjy98wLySGqxdMrdBDZO4dwCHTOojROxaxPLW9rmroUBtgufYQui_tgJ8QAI2EOEXIU0dO8bRNCoDgTu9X8E6f0cuY4ugNR1-n0-eV7CahkcmzXACLm8i8Dm33ZTsxf5sI-fsTAGNk4HtvYgXm5wRG4VJj_M0o5nOHs","p":"9inwA4qS3eJfrF1-_dmYAamfK5yhEeA62109oBrSgv_QEyVF1MbSyZmGt9pTvGjwieg5XsCVVDgFBuXGnywcsGpBl5ILDP-rJF69fYW6zwfKU8c7c5r2BTIrjZRO-Q23VxgX2cGjjmdV8tSS-FzyZ3mGUtYKGRR4E1r8Mpi8w-xcXkjX5Uy31f93Y_nEkKEnmHkEbu1CVBEJaABCt7eqMB5dbZ67B7XWtMklpEzoiESfBrqlMRP4us1S9Jgj6msMA3Ghj9lj1x4ux2x1wqGKD3WXIvihaFTU4myBSyl9yhpC1zKm_eE_E5mEwo6VS5o7aRfnRRgqZicSEjLZg_IzCQ","q":"8-vXnFzh4Nr8jDVIWYmPiJULIBr4_ONCEQ5VxYTj_XW-jQGWrO9G9CBkwSVDOrHJ2bynddypqozKZxiloTqcdrZodS_9CPleuuP6Vli3v2E0aMziSlCiF8yyhY75-zgOei5nQ0fNsw5a6hHM2hkzjT4vghBIxkOyRSfSFkQd8wKzXtau9DKc0N7QetH5YvXG79bBY2XYMsbylQK3eEcP5oW1-DTMQd_bdVR1ILapayNMs6aGb2BF2hIlHIdsmy1wv6T3EMEytY8HOvhyx4vQ9nD6MXBBO6m9rKnEtIEt4XUQQW_sdhCdleyZ16eaYQidq7WYKIp2PR4YrRmdoIScYw","qi":"sJpqEXL-LN5jvAA3FZMArmXZhlB730QkuPrk7IvOnEhABSUVXNCV3VFPotfyB_NAjd5lrHHpxemSo9JES8aBadlZyVJsS3i64UQTQzyyUw9NAhrzWqBUBHfLg9qP1utCMzw90m5lVuYBiaD4yfcpoIe710tIovJ3tY_GyiG9pKQMrN_SOBAns65yWBGNgjih_MSp-OCMJJ2Peu-JexRBNUqV-rzXcT6nQs4_Rrkmhat2ltrB-8OMXQemigVWmV0wK_--A_uBtFX0-p-b4JK8ipRPc-8CHOv-v8NCt9ATQgySFTLfY9J9yjS5uH2KK35s-bJQfpVHZ5_WvZQMXGII-A"}`)
	viper.SetDefault("ArRequestTimeout", "120s")
	viper.SetDefault("ListenerQueueSize", "50")
	viper.SetDefault("PollerNetworkInfoTimeout", "2s")
	viper.SetDefault("PollerWorkerPoolSize", "10")
	viper.SetDefault("StoreBatchSize", "50")
	viper.SetDefault("StoreMaxTimeInQueue", "1s")
	viper.SetDefault("StoreMaxTimeBetweenReconnects", "1s")
}

func Default() (config *Config) {
	config, _ = Load("")
	return
}

// Load configuration from file and env
func Load(filename string) (config *Config, err error) {
	viper.SetConfigType("json")
	viper.AutomaticEnv()         // read in environment variables that match
	viper.SetEnvPrefix("syncer") // will be uppercased automatically
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	setDefaults()

	// Empty filename means we use default values
	if filename != "" {
		var content []byte
		/* #nosec */
		content, err = os.ReadFile(filename)
		if err != nil {
			return nil, err
		}

		err = viper.ReadConfig(bytes.NewBuffer(content))
		if err != nil {
			return nil, err
		}
	}

	err = viper.Unmarshal(&config)

	return
}
