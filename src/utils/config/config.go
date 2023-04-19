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

	Arweave               Arweave
	PeerMonitor           PeerMonitor
	TransactionDownloader TransactionDownloader
	NetworkMonitor        NetworkMonitor
	Syncer                Syncer
	Bundler               Bundler
	Bundlr                Bundlr
	Checker               Checker
	Database              Database
	Contract              Contract
	Redis                 Redis
	AppSync               AppSync
}

func setDefaults() {
	viper.SetDefault("RESTListenAddress", ":3333")
	viper.SetDefault("LogLevel", "DEBUG")
	viper.SetDefault("StopTimeout", "30s")

	setArweaveDefaults()
	setPeerMonitorDefaults()
	setTransactionDownloaderDefaults()
	setNetworkMonitorDefaults()
	setSyncerDefaults()
	setBundlerDefaults()
	setBundlrDefaults()
	setCheckerDefaults()
	setDatabaseDefaults()
	setContractDefaults()
	setRedisDefaults()
	setAppSyncDefaults()
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
