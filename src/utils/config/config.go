package config

import (
	"bytes"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config stores global configuration
type Config struct {
	// SQL Database
	DBPort     string
	DBHost     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string

	// Logging
	LogLevel string
	LogPath  string

	// Arsync
	ArNodeUrl               string
	ArConcurrentConnections int
	ArStableDistance        int64
	ListenerQueueSize       int
}

func setDefaults() {
	viper.SetDefault("DBPort", "7654")
	viper.SetDefault("DBHost", "127.0.0.1")
	viper.SetDefault("DBUser", "postgres")
	viper.SetDefault("DBPassword", "postgres")
	viper.SetDefault("DBName", "redstone")
	viper.SetDefault("DBSSLMode", "disable")
	viper.SetDefault("LogLevel", "DEBUG")
	viper.SetDefault("LogPath", "")
	viper.SetDefault("ArNodeUrl", "https://arweave.net")
	viper.SetDefault("ArConcurrentConnections", "50")
	viper.SetDefault("ArStableDistance", "15")
	viper.SetDefault("ListenerQueueSize", "50")
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
