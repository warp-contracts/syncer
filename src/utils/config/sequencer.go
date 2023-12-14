package config

import (
	"time"

	"github.com/spf13/viper"
)

type Sequencer struct {
	// List of urls that can be used, order matters
	Urls []string

	// Time limit for requests. The timeout includes connection time, any
	// redirects, and reading the response body
	RequestTimeout time.Duration

	// Maximum amount of time a dial will wait for a connect to complete.
	DialerTimeout time.Duration

	// Interval between keep-alive probes for an active network connection.
	DialerKeepAlive time.Duration

	// Maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.
	IdleConnTimeout time.Duration

	// Maximum amount of time waiting to wait for a TLS handshake
	TLSHandshakeTimeout time.Duration

	// Time in which max num of requests is enforced
	LimiterInterval time.Duration

	// Max num requests to particular host per interval
	LimiterBurstSize int
}

func setSequencerDefaults() {
	viper.SetDefault("Sequencer.Urls", []string{"http://sequencer-0.devnet.warp.cc", "http://sequencer-1.devnet.warp.cc", "http://sequencer-2.devnet.warp.cc"})
	viper.SetDefault("Sequencer.RequestTimeout", "30s")
	viper.SetDefault("Sequencer.DialerTimeout", "30s")
	viper.SetDefault("Sequencer.DialerKeepAlive", "15s")
	viper.SetDefault("Sequencer.IdleConnTimeout", "31s")
	viper.SetDefault("Sequencer.TLSHandshakeTimeout", "10s")
	viper.SetDefault("Sequencer.LimiterInterval", "24h")
	viper.SetDefault("Sequencer.LimiterBurstSize", "10000000000000000")
}
