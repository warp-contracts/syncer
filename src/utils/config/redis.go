package config

import (
	"time"

	"github.com/spf13/viper"
)

type Redis struct {
	Port     uint16
	Host     string
	User     string
	Password string
	DB       int

	// TLS configuration
	ClientKey  string
	ClientCert string
	CaCert     string

	// Connection configuration
	MinIdleConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

func setRedisDefaults() {
	viper.SetDefault("Redis.Port", "6379")
	viper.SetDefault("Redis.Host", "localhost")
	viper.SetDefault("Redis.User", "")
	viper.SetDefault("Redis.Password", "password")
	viper.SetDefault("Redis.DB", "0")
	viper.SetDefault("Redis.MinIdleConns", "1")
	viper.SetDefault("Redis.MaxIdleConns", "5")
	viper.SetDefault("Redis.ConnMaxIdleTime", "10m")
	viper.SetDefault("Redis.MaxOpenConns", "15")
	viper.SetDefault("Redis.ConnMaxLifetime", "1h")
}
