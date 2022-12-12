package cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(serverCmd)
}

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Listen for changes from Arweave nodes and save to the database",
	Run: func(cmd *cobra.Command, args []string) {
		<-ctx.Done()
	},
}
