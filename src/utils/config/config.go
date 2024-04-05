package config

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

const MAX_SLICE_LEN = 10

// Config stores global configuration
type Config struct {
	// Is development mode on
	IsDevelopment bool

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
	Sender                Sender
	Bundlr                Bundlr
	Sequencer             Sequencer
	Checker               Checker
	Database              Database
	ReadOnlyDatabase      Database
	Contract              Contract
	Redis                 []Redis
	AppSync               AppSync
	Forwarder             Forwarder
	Relayer               Relayer
	Gateway               Gateway
	Profiler              Profiler
	Interactor            Interactor
	Evolver               Evolver
	WarpySyncer           WarpySyncer
}

func setDefaults() {
	viper.SetDefault("IsDevelopment", "false")
	viper.SetDefault("RESTListenAddress", ":7777")
	viper.SetDefault("LogLevel", "DEBUG")
	viper.SetDefault("StopTimeout", "30s")

	setForwarderDefaults()
	setArweaveDefaults()
	setPeerMonitorDefaults()
	setTransactionDownloaderDefaults()
	setNetworkMonitorDefaults()
	setSyncerDefaults()
	setBundlerDefaults()
	setSenderDefaults()
	setBundlrDefaults()
	setCheckerDefaults()
	setDatabaseDefaults()
	setReadOnlyDatabaseDefaults()
	setContractDefaults()
	setRedisDefaults()
	setAppSyncDefaults()
	setRelayerDefaults()
	setGatewayDefaults()
	setProfilerDefaults()
	setInteractorDefaults()
	setSequencerDefaults()
	setEvolverDefaults()
	setWarpySyncerDefaults()
}

func Default() (config *Config) {
	config, _ = Load("")
	return
}

func IsIndex(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func BindEnv(path []string, val reflect.Value) {
	if val.Kind() == reflect.Slice {
		_, ok := val.Interface().([]Redis)
		if ok {
			for i := 0; i < MAX_SLICE_LEN; i++ {
				newPath := make([]string, len(path))
				copy(newPath, path)
				newPath = append(newPath, fmt.Sprintf("%d", i))
				BindEnv(newPath, reflect.ValueOf(Redis{}))
			}
		} else {
			// Slice of base types
			key := strings.ToLower(strings.Join(path, "."))
			env := "SYNCER_" + strcase.ToScreamingSnake(strings.Join(path, "_"))
			err := viper.BindEnv(key, env)
			if err != nil {
				panic(err)
			}
		}
	} else if val.Kind() != reflect.Struct {
		// Base types
		// key := strings.ToLower(strings.Join(path, "."))
		key := path[0]
		for _, p := range path[1:] {
			if IsIndex(p) {
				key += "[" + p + "]"
				// key += "." + p
			} else {
				key += "." + p
			}
		}

		env := "SYNCER_" + strcase.ToScreamingSnake(strings.Join(path, "_"))
		err := viper.BindEnv(key, env)
		if err != nil {
			panic(err)
		}
	} else {
		// Iterates over struct fields
		for i := 0; i < val.NumField(); i++ {
			newPath := make([]string, len(path))
			copy(newPath, path)
			newPath = append(newPath, val.Type().Field(i).Name)
			BindEnv(newPath, val.Field(i))
		}
	}
}

func getSliceLength(key string) int {
	var max int
	for viperKey := range viper.AllSettings() {
		var idx int
		// var rest string
		_, err := fmt.Sscanf(viperKey, key+"[%d]", &idx)
		if err != nil {
			continue
		}
		idx += 1
		if idx > max {
			max = idx
		}
	}
	return max
}

func defaultDecoderConfig(output interface{}) *mapstructure.DecoderConfig {
	c := &mapstructure.DecoderConfig{
		Metadata:         nil,
		Result:           output,
		WeaklyTypedInput: true,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	}
	return c
}

// Load configuration from file and env
func Load(filename string) (config *Config, err error) {
	viper.SetConfigType("json")
	// viper.SetTypeByDefaultValue(true)

	setDefaults()

	// Visits every field and registers upper snake case ENV name for it
	// Works with embedded structs
	BindEnv([]string{}, reflect.ValueOf(Config{}))

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

	config = new(Config)
	err = viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}

	err = unmarshalRedis(config)
	if err != nil {
		return nil, err
	}

	return
}
