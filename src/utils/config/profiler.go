package config

import (
	"github.com/spf13/viper"
)

type Profiler struct {
	// Are profiling endpoints registered
	Enabled bool

	//BlockProfileRate
	BlockProfileRate int
}

func setProfilerDefaults() {
	viper.SetDefault("Profiler.Enabled", "true")
	viper.SetDefault("Profiler.BlockProfileRate", "50")
}
