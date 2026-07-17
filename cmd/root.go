// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "nui",
	Short: "nui CLI",
	Long:  "nui is a CLI application with an optional web UI.",
}

// Version is the CLI release version (set from main via SetVersion).
var Version = "dev"

// SetVersion wires the release version into the root command for --version.
func SetVersion(v string) {
	Version = v
	rootCmd.Version = v
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
