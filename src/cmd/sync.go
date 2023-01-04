package cmd

import (
	"syncer/src/sync"
	"syncer/src/utils/logger"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "sync",
	Short: "Listen for changes from Arweave nodes and save to the database",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		sync, err := sync.NewController(conf)
		if err != nil {
			return
		}

		sync.Start()

		select {
		case <-sync.Ctx.Done():
		case <-applicationCtx.Done():
		}

		sync.StopWait()

		return
	},
	PostRunE: func(cmd *cobra.Command, args []string) (err error) {
		log := logger.NewSublogger("root-cmd")
		log.Debug("Finished sync command")
		applicationCtxCancel()
		return
	},
}
