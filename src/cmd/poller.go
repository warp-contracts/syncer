package cmd

import (
	"syncer/src/poller"
	"syncer/src/utils/logger"

	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "poller",
	Short: "Listen for changes from Arweave nodes and save to the database",
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		poller, err := poller.NewPoller(ctx, conf)
		if err != nil {
			return
		}

		poller.Start()

		<-ctx.Done()

		return
	},
	PostRunE: func(cmd *cobra.Command, args []string) (err error) {
		log := logger.NewSublogger("root-cmd")
		log.Debug("Finished poll command")
		return
	},
}
