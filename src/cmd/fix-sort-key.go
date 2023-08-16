package cmd

import (
	"github.com/warp-contracts/syncer/src/fixsortkey"
	"github.com/warp-contracts/syncer/src/utils/logger"

	"github.com/spf13/cobra"
)

func init() {
	fixSortKey.PersistentFlags().Uint64Var(&startBlockHeight, "start", 0, "Start block height")
	fixSortKey.PersistentFlags().Uint64Var(&stopBlockHeight, "stop", 0, "Stop block height")
	RootCmd.AddCommand(fixSortKey)
}

var (
	fixSortKey = &cobra.Command{
		Use:   "fix_sort_key",
		Short: "Rewrites sort key and last sort key for all interactions",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			controller, err := fixsortkey.NewController(conf, startBlockHeight, stopBlockHeight)
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
			log.Debug("Finished fix_sort_key command")
			applicationCtxCancel()
			return
		},
	}
)
