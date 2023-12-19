package config

import (
	"time"

	"github.com/spf13/viper"
)

type Interactor struct {
	// Contract id tag for the generated data items
	GeneratorContractId string

	// How often to generate data item
	GeneratorInterval time.Duration

	// Key/wallet used to sign the generated data item
	GeneratorEthereumKey string

	// How often to check for processed data items in database
	CheckerInterval time.Duration

	// Number of workers that will be sending data items to sequencer
	SenderWorkerPoolSize int

	// Size of the queue that will be used to send data items to sequencer
	SenderWorkerQueueSize int
}

func setInteractorDefaults() {
	viper.SetDefault("Interactor.GeneratorContractId", "dev-interactor-monitor-contract-00000000000")
	viper.SetDefault("Interactor.GeneratorInterval", "30s")
	viper.SetDefault("Interactor.GeneratorEthereumKey", "0xdbf7711298c3c8c9fc68713c4bb9021ef4e219f1a984fc298bbac20e38bcc213")
	viper.SetDefault("Interactor.CheckerInterval", "30s")
	viper.SetDefault("Interactor.SenderWorkerPoolSize", "1")
	viper.SetDefault("Interactor.SenderWorkerQueueSize", "1")
}
