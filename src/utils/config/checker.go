package config

import (
	"time"

	"github.com/spf13/viper"
)

type Checker struct {
	// How often to check for new network info
	Interval time.Duration

	// Minimal number of blocks to wait before checking the bundle
	MinConfirmationBlocks int64

	// Number of bundles to confirm in one run.
	MaxBundlesPerRun int

	// Number of workers that check bundles in parallel
	WorkerPoolSize int

	// Size of the queue for workers
	WorkerQueueSize int
}

func setCheckerDefaults() {
	viper.SetDefault("Checker.Interval", "30s")
	viper.SetDefault("Checker.MinConfirmationBlocks", "52")
	viper.SetDefault("Checker.MaxBundlesPerRun", "50")
	viper.SetDefault("Checker.WorkerPoolSize", "50")
	viper.SetDefault("Checker.WorkerQueueSize", "150")
}
