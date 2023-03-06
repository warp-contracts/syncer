package config

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/spf13/viper"
)

// Config stores global configuration
type Config struct {
	// REST API address. API used for monitoring etc.
	RESTListenAddress string

	// Maximum time Syncer will be closing before stop is forced.
	StopTimeout time.Duration

	// Logging level
	LogLevel string

	// URL to arweave.net
	ArNodeUrl string

	// Time limit for requests. The timeout includes connection time, any
	// redirects, and reading the response body
	ArRequestTimeout time.Duration

	// Miminum time a peer needs to answer in order to be considered responsive.
	// This should be much smaller than request timeout
	ArCheckPeerTimeout time.Duration

	// Maximum amount of time a dial will wait for a connect to complete.
	ArDialerTimeout time.Duration

	// Interval between keep-alive probes for an active network connection.
	ArDialerKeepAlive time.Duration

	// Maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.
	ArIdleConnTimeout time.Duration

	// Maximum amount of time waiting to wait for a TLS handshake
	ArTLSHandshakeTimeout time.Duration

	// https://ar-io.zendesk.com/hc/en-us/articles/7595655106971-arweave-net-Rate-Limits
	// Time in which max num of requests is enforced
	ArLimiterInterval time.Duration

	// Max num requests to particular peer per interval
	ArLimiterBurstSize int

	// Limit is a float numbef = max frequency per second. Whenever a HTTP 429 Too Many Requests is received we multiply limit by this factor.
	// This way even if the limit is set too high eventually it'll get small enough.
	ArLimiterDecreaseFactor float64

	// How often limiters get decreased. This timeout won't allow sudden burst to decrease the limit too much
	ArLimiterDecreaseInterval time.Duration

	// Received Arweave transactions converted to interactions and temporarily stored in the channel
	// Normally interactions are passed to the Store right away, but if Store is in the middle of transaction it's not receiving data.
	// So this capacity should account for interactions that may appear during a few second window when the previous batch is inserted to the database.
	ListenerQueueSize int

	// Minimum amount of confirmations (blocks on top of the given block) that are required to consider
	// a given block as confirmed (i.e. not being a fork)
	ListenerRequiredConfirmationBlocks int64

	// URL of the node we're using to get the current block height.
	// It's the Warp's Gateway URL to avoid race conditions
	ListenerNetworkInfoNodeUrl string

	// Time between requests to the Warp's Gateway for network info
	ListenerPeriod time.Duration

	// Time between failed retries to download transaction
	ListenerRetryFailedTransactionDownloadInterval time.Duration

	// Number of workers that download the transactions
	ListenerNumWorkers int

	// Maximum time a peer is blacklisted.
	// Even after this duration is over it may take some time for the peer to be re-checked
	PeerMonitorMaxTimeBlacklisted time.Duration

	// Maximum number of peers that can be removed from the blacklist
	// Peers that are blacklisted longer than `PeerMonitorMaxTimeBlacklisted` will get eventually re-used
	// To avoid spikes we can only remove at most this many peers from the blacklist in one try
	PeerMonitorMaxPeersRemovedFromBlacklist int

	// Time between sending monitoring requests to peers
	// Peers are downloaded from the arweave API and checked in parallel by couple of workers
	PeerMonitorPeriod time.Duration

	// Max number of peers that can be used for retrying requests
	PeerMonitorMaxPeers int

	// Number of workers that check peers in parallel
	PeerMonitorNumWorkers int

	// Num of Interactions that are stored in the Store
	// before being inserted into the database in one db transaction and batch.
	StoreBatchSize int

	// After this time all Interactions in Store will be inserted to the database.
	// This is to avoid keeping them in the service for too long when waiting to fill the batch.
	StoreMaxTimeInQueue time.Duration

	// Max time between failed retries to save data.
	StoreMaxBackoffInterval time.Duration

	// How often is IM pooling the database
	InteractionManagerInterval time.Duration

	// How long does it wait for the query response
	InteractionManagerTimeout time.Duration

	// Maksimum number of requests run in parallel
	InteractionManagerMaxParallelQueries int

	Bundler  Bundler
	Bundlr   Bundlr
	Checker  Checker
	Database Database
}

func setDefaults() {
	viper.SetDefault("RESTListenAddress", ":3333")
	viper.SetDefault("LogLevel", "DEBUG")

	viper.SetDefault("StopTimeout", "30s")

	viper.SetDefault("ArNodeUrl", "https://arweave.net")
	viper.SetDefault("ArRequestTimeout", "30s")
	viper.SetDefault("ArCheckPeerTimeout", "1s")
	viper.SetDefault("ArDialerTimeout", "30s")
	viper.SetDefault("ArDialerKeepAlive", "15s")
	viper.SetDefault("ArIdleConnTimeout", "31s")
	viper.SetDefault("ArTLSHandshakeTimeout", "10s")
	viper.SetDefault("ArLimiterInterval", "1s")
	viper.SetDefault("ArLimiterBurstSize", "15")
	viper.SetDefault("ArLimiterDecreaseFactor", "1.0")
	viper.SetDefault("ArLimiterDecreaseInterval", "2m")

	viper.SetDefault("ListenerQueueSize", "1")
	viper.SetDefault("ListenerNetworkInfoNodeUrl", "https://gateway.warp.cc/gateway/arweave")
	viper.SetDefault("ListenerPeriod", "1s")
	viper.SetDefault("ListenerRetryFailedTransactionDownloadInterval", "10s")
	viper.SetDefault("ListenerRequiredConfirmationBlocks", "10")
	viper.SetDefault("ListenerNumWorkers", "50")

	viper.SetDefault("PeerMonitorMaxTimeBlacklisted", "30m")
	viper.SetDefault("PeerMonitorMaxPeersRemovedFromBlacklist", "5")
	viper.SetDefault("PeerMonitorPeriod", "10m")
	viper.SetDefault("PeerMonitorMaxPeers", "15")
	viper.SetDefault("PeerMonitorNumWorkers", "40")

	viper.SetDefault("StoreBatchSize", "500")
	viper.SetDefault("StoreMaxTimeInQueue", "10s")
	viper.SetDefault("StoreMaxBackoffInterval", "30s")

	setBundlerDefaults()
	setBundlrDefaults()
	setCheckerDefaults()
	setDatabaseDefaults()
}

func Default() (config *Config) {
	config, _ = Load("")
	return
}

func BindEnv(path []string, val reflect.Value) {
	if val.Kind() != reflect.Struct {
		key := strings.ToLower(strings.Join(path, "."))
		env := "SYNCER_" + strcase.ToScreamingSnake(strings.Join(path, "_"))
		err := viper.BindEnv(key, env)
		if err != nil {
			panic(err)
		}
	} else {
		for i := 0; i < val.NumField(); i++ {
			newPath := make([]string, len(path))
			copy(newPath, path)
			newPath = append(newPath, val.Type().Field(i).Name)
			BindEnv(newPath, val.Field(i))
		}
	}
}

// Load configuration from file and env
func Load(filename string) (config *Config, err error) {
	viper.SetConfigType("json")

	// Visits every field and registers upper snake case ENV name for it
	// Works with embedded structs
	BindEnv([]string{}, reflect.ValueOf(Config{}))

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
