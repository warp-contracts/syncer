package config

import (
	"github.com/spf13/viper"
)

type Gateway struct {
	// REST API address. API used for monitoring etc.
	RESTListenAddress string
}

func setGatewayDefaults() {
	viper.SetDefault("Gateway.RESTListenAddress", "0.0.0.0:4000")
}
