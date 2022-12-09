package cmd

import (
	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:           "syncer",
		Short:         "Tool listening for changes from Arweave nodes",
		SilenceErrors: true,
	}
)
