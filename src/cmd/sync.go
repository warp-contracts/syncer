package cmd

import (
	"github.com/warp-contracts/syncer/src/sync"
	"github.com/warp-contracts/syncer/src/utils/logger"

	"github.com/spf13/cobra"
)

func init() {
	syncCmd.PersistentFlags().Uint64Var(&startBlockHeight, "start", 0, "Start block height")
	syncCmd.PersistentFlags().Uint64Var(&stopBlockHeight, "stop", 0, "Stop block height")
	syncCmd.PersistentFlags().BoolVar(&replaceExistingData, "DANGEROUS_replace_existing_data", false, "Replace data that is already in the database. Default: false")
	RootCmd.AddCommand(syncCmd)
}

var (
	syncCmd = &cobra.Command{
		Use:   "sync",
		Short: "Save L1 interactions to the database",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			controller, err := sync.NewController(conf, startBlockHeight, stopBlockHeight, replaceExistingData)
			if err != nil {
				return
			}

			err = controller.Start()
			if err != nil {
				return
			}

			select {
			case <-controller.CtxRunning.Done():
			case <-applicationCtx.Done():
			}

			controller.StopWait()

			return
		},
		PostRunE: func(cmd *cobra.Command, args []string) (err error) {
			log := logger.NewSublogger("root-cmd")
			log.Debug("Finished sync command")
			applicationCtxCancel()
			return
		},
	}
)
