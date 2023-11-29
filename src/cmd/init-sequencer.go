package cmd

import (
	"errors"
	"github.com/spf13/cobra"

	"github.com/warp-contracts/syncer/src/initsequencer"
	"github.com/warp-contracts/syncer/src/utils/logger"
)

func init() {
	initSequencerCmd.PersistentFlags().StringVar(&sequencerRepoPath, "sequencer-repo-path", "", "The path to the sequencer repository")
	initSequencerCmd.PersistentFlags().StringVar(&env, "env", "", "Environment name")
	RootCmd.AddCommand(initSequencerCmd)
}

var sequencerRepoPath string
var env string

var initSequencerCmd = &cobra.Command{
	Use:   "init_sequencer",
	Short: "Fetching the data required to start a decentralized sequencer.",
	RunE: func(cmd *cobra.Command, args []string) (err error) {

		if len(sequencerRepoPath) == 0 {
			return errors.New("No path to the sequencer repository provided")
		}

		if len(env) == 0 {
			return errors.New("No environment name provided")
		}

		controller, err := initsequencer.NewController(conf, sequencerRepoPath, env)
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
		log := logger.NewSublogger("init-sequencer-cmd")
		log.Debug("Finished init_sequencer command")
		applicationCtxCancel()
		return
	},
}
