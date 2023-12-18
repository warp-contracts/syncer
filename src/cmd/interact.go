package cmd

import (
	"github.com/warp-contracts/syncer/src/interact"
	"github.com/warp-contracts/syncer/src/utils/logger"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(interactCmd)
}

var interactCmd = &cobra.Command{
	Use:   "interact",
	Short: "Send data items to Warp's sequencer and check if they get inserted into the database",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		controller, err := interact.NewController(conf)
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
		log.Debug("Finished interact command")
		applicationCtxCancel()
		return
	},
}
