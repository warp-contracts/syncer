package config

import (
	"time"

	"github.com/spf13/viper"
)

type Bundlr struct {
	// List of urls that can be used, order matters
	Urls []string

	// Time limit for requests. The timeout includes connection time, any
	// redirects, and reading the response body
	RequestTimeout time.Duration

	// Miminum time a peer needs to answer in order to be considered responsive.
	// This should be much smaller than request timeout
	CheckPeerTimeout time.Duration

	// Maximum amount of time a dial will wait for a connect to complete.
	DialerTimeout time.Duration

	// Interval between keep-alive probes for an active network connection.
	DialerKeepAlive time.Duration

	// Maximum amount of time an idle (keep-alive) connection will remain idle before closing itself.
	IdleConnTimeout time.Duration

	// Maximum amount of time waiting to wait for a TLS handshake
	TLSHandshakeTimeout time.Duration

	// https://ar-io.zendesk.com/hc/en-us/articles/7595655106971-arweave-net-Rate-Limits
	// Time in which max num of requests is enforced
	LimiterInterval time.Duration

	// Max num requests to particular peer per interval
	LimiterBurstSize int
}

func setBundlrDefaults() {
	viper.SetDefault("Bundlr.Urls", []string{"https://node1.bundlr.network", "https://node1.bundlr.network"})
	viper.SetDefault("Bundlr.RequestTimeout", "30s")
	viper.SetDefault("Bundlr.CheckPeerTimeout", "1s")
	viper.SetDefault("Bundlr.DialerTimeout", "30s")
	viper.SetDefault("Bundlr.DialerKeepAlive", "15s")
	viper.SetDefault("Bundlr.IdleConnTimeout", "31s")
	viper.SetDefault("Bundlr.TLSHandshakeTimeout", "10s")
	viper.SetDefault("Bundlr.LimiterInterval", "500ms")
	viper.SetDefault("Bundlr.LimiterBurstSize", "7")
}
