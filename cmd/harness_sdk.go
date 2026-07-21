// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"

	"nui/harness-sdk"

	"github.com/spf13/cobra"
)

var harnessSDKCmd = &cobra.Command{
	Use:   "harness-sdk",
	Short: "Manage the Python harness SDK under ~/.nui/harness-sdk/",
}

var harnessSDKReinstallCmd = &cobra.Command{
	Use:   "reinstall",
	Short: "Reinstall harness-sdk files from the nui binary",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := harnesssdk.ReinstallDir()
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stdout, "Reinstalled %d harness-sdk files to %s\n", len(harnesssdk.FileNames), dir)
		return nil
	},
}

func init() {
	harnessSDKCmd.AddCommand(harnessSDKReinstallCmd)
	rootCmd.AddCommand(harnessSDKCmd)
}
