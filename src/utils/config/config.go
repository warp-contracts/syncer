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

	// Logging
	LogLevel string
	LogPath  string

	ArNodeUrl               string
	ArConcurrentConnections int
	ArStableDistance        int64

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
	viper.SetDefault("LogLevel", "DEBUG")
	viper.SetDefault("LogPath", "")
	viper.SetDefault("ArNodeUrl", "https://arweave.net")
	viper.SetDefault("ArConcurrentConnections", "50")
	viper.SetDefault("ArStableDistance", "15")
	viper.SetDefault("ListenerQueueSize", "50")
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
