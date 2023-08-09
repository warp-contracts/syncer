package config

import (
	"time"

	"github.com/spf13/viper"
)

type Database struct {
	Port              uint16
	Host              string
	User              string
	Password          string
	Name              string
	SslMode           string
	PingTimeout       time.Duration
	ClientKey         string
	ClientKeyPath     string
	ClientCert        string
	ClientCertPath    string
	CaCert            string
	CaCertPath        string
	MigrationUser     string
	MigrationPassword string

	// Connection configuration
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
}

func setDatabaseDefaults() {
	viper.SetDefault("Database.Port", "7654")
	viper.SetDefault("Database.Host", "127.0.0.1")
	viper.SetDefault("Database.User", "postgres")
	viper.SetDefault("Database.Password", "postgres")
	viper.SetDefault("Database.Name", "warp")
	viper.SetDefault("Database.SslMode", "disable")
	viper.SetDefault("Database.PingTimeout", "15s")
	viper.SetDefault("Database.MigrationUser", "postgres")
	viper.SetDefault("Database.MigrationPassword", "postgres")
	viper.SetDefault("Database.MaxOpenConns", "50")
	viper.SetDefault("Database.MaxIdleConns", "30")
	viper.SetDefault("Database.ConnMaxIdleTime", "10m")
	viper.SetDefault("Database.ConnMaxLifetime", "2h")
}

func setReadOnlyDatabaseDefaults() {
	viper.SetDefault("ReadOnlyDatabase.Port", "7654")
	viper.SetDefault("ReadOnlyDatabase.Host", "127.0.0.1")
	viper.SetDefault("ReadOnlyDatabase.User", "postgres")
	viper.SetDefault("ReadOnlyDatabase.Password", "postgres")
	viper.SetDefault("ReadOnlyDatabase.Name", "warp")
	viper.SetDefault("ReadOnlyDatabase.SslMode", "disable")
	viper.SetDefault("ReadOnlyDatabase.PingTimeout", "15s")
	viper.SetDefault("ReadOnlyDatabase.MigrationUser", "postgres")
	viper.SetDefault("ReadOnlyDatabase.MigrationPassword", "postgres")
	viper.SetDefault("ReadOnlyDatabase.MaxOpenConns", "50")
	viper.SetDefault("ReadOnlyDatabase.MaxIdleConns", "30")
	viper.SetDefault("ReadOnlyDatabase.ConnMaxIdleTime", "10m")
	viper.SetDefault("ReadOnlyDatabase.ConnMaxLifetime", "2h")
}
