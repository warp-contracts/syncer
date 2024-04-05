package cmd

import (
	"github.com/warp-contracts/syncer/src/utils/logger"
	"github.com/warp-contracts/syncer/src/warpy_sync"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(warpy_syncCmd)
}

var warpy_syncCmd = &cobra.Command{
	Use:   "warpy_sync",
	Short: "Get new RedStone transactions from external chains and send interactions to Warpy",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		controller, err := warpy_sync.NewController(conf)
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
		log.Debug("Finished warpy_sync command")
		applicationCtxCancel()
		return
	},
}
