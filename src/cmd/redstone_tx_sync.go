package cmd

import (
	"github.com/warp-contracts/syncer/src/redstone_tx_sync"
	"github.com/warp-contracts/syncer/src/utils/logger"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(redstone_tx_syncCmd)
}

var redstone_tx_syncCmd = &cobra.Command{
	Use:   "redstone_tx_sync",
	Short: "Get new RedStone transactions from external chains and send interactions to Warpy",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		controller, err := redstone_tx_sync.NewController(conf)
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
		log.Debug("Finished redstone_tx_sync command")
		applicationCtxCancel()
		return
	},
}
