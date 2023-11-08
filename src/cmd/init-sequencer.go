package cmd

import (
	"errors"
	"github.com/spf13/cobra"

	"github.com/warp-contracts/syncer/src/initsequencer"
	"github.com/warp-contracts/syncer/src/utils/logger"
)

func init() {
	initSequencerCmd.PersistentFlags().StringVar(&warpInternalRepoPath, "warp-internal-repo-path", "", "The path to the warp internal repository")
	initSequencerCmd.PersistentFlags().StringVar(&sequencerRepoPath, "sequencer-repo-path", "", "The path to the sequencer repository")
	RootCmd.AddCommand(initSequencerCmd)
}

var warpInternalRepoPath string
var sequencerRepoPath string

var initSequencerCmd = &cobra.Command{
	Use:   "init_sequencer",
	Short: "Fetching the data required to start a decentralized sequencer.",
	RunE: func(cmd *cobra.Command, args []string) (err error) {

		if len(warpInternalRepoPath) == 0 {
			return errors.New("No path to the warp internal repository provided")
		}

		if len(sequencerRepoPath) == 0 {
			return errors.New("No path to the sequencer repository provided")
		}

		controller, err := initsequencer.NewController(conf, warpInternalRepoPath, sequencerRepoPath)
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
