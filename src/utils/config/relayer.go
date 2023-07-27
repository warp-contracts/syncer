package config

import (
	"github.com/spf13/viper"
)

type Relayer struct {
	// Websocket url of the Warp's sequencer
	SequencerUrl string

	// How many incomming events should be stored in channel
	SequencerQueueSize int
}

func setRelayerDefaults() {
	viper.SetDefault("Relayer.SequencerUrl", "tcp://127.0.0.1:26657")
	viper.SetDefault("Relayer.SequencerQueueSize", "100")
}
