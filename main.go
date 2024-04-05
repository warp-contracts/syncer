// Package main is just the application entry point
package main

import (
	"fmt"
	"os"

	"github.com/warp-contracts/syncer/src/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %+v\n", err)
		os.Exit(1)
	}
}
