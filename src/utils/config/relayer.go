package config

import (
	"github.com/spf13/viper"
)

type Relayer struct {
	// Websocket url of the Warp's sequencer
	SequencerUrl string
}

func setRelayerDefaults() {
	viper.SetDefault("Relayer.SequencerUrl", "wss://localhost:8080")
}
