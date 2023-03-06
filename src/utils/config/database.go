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
	ClientCert        string
	CaCert            string
	MigrationUser     string
	MigrationPassword string
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
}
