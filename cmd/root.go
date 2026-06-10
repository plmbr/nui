// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "loop",
	Short: "Loop CLI",
	Long:  "Loop is a CLI application with an optional web UI.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
