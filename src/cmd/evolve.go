package cmd

import (
	"github.com/warp-contracts/syncer/src/evolve"
	"github.com/warp-contracts/syncer/src/utils/logger"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(evolveCmd)
}

var evolveCmd = &cobra.Command{
	Use:   "evolve",
	Short: "Indexing new contract sources based on evolve interactions",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		controller, err := evolve.NewController(conf)
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
		log.Debug("Finished evolve command")
		applicationCtxCancel()
		return
	},
}
