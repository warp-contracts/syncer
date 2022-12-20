package cmd

import (
	"context"
	"os"
	"os/signal"
	"syncer/src/utils/common"
	"syncer/src/utils/config"
	"syncer/src/utils/logger"
	"syscall"

	"github.com/spf13/cobra"
)

var (
	RootCmd = &cobra.Command{
		Use:   "syncer",
		Short: "Tool listening for changes from Arweave nodes",

		// All child commands will use this
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			// Setup a context that gets cancelled upon SIGINT
			applicationCtx, applicationCtxCancel = context.WithCancel(context.Background())

			signalChannel = make(chan os.Signal, 1)
			signal.Notify(signalChannel, os.Interrupt, syscall.SIGTERM)
			go func() {
				select {
				case <-signalChannel:
					applicationCtxCancel()
				case <-applicationCtx.Done():
				}
			}()

			// Load configuration
			conf, err = config.Load(cfgFile)
			if err != nil {
				return
			}
			applicationCtx = common.SetConfig(applicationCtx, conf)

			// Setup logging
			err = logger.Init(conf)
			if err != nil {
				return
			}
			return
		},

		// Run after all commands
		PersistentPostRunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				signal.Stop(signalChannel)
				applicationCtxCancel()
			}()
			log := logger.NewSublogger("root-cmd")
			<-applicationCtx.Done()
			log.Debug("Finished")
			return
		},
		SilenceErrors: true,
	}

	// Configuration
	conf    *config.Config
	cfgFile string

	// Application context.
	// As soon as application ctx gets canceled the app stars to shutdown
	// It shuts down gracefully, finishes processing within a predefined timeout
	applicationCtx       context.Context
	applicationCtxCancel context.CancelFunc

	// Signals from the OS
	signalChannel chan os.Signal
)

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "configuration file path")
}
