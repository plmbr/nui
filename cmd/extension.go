// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"

	"nui/internal/extensions"

	"github.com/spf13/cobra"
)

var extensionCmd = &cobra.Command{
	Use:   "extension",
	Short: "Manage nui extensions",
}

var extensionAddCmd = &cobra.Command{
	Use:   "add [url-or-path]",
	Short: "Install an extension from a git URL, directory, or zip file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := extensions.Install(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Installed extension %q\n", name)
		return nil
	},
}

var extensionRemoveCmd = &cobra.Command{
	Use:   "remove [ext-id]",
	Short: "Remove an installed extension",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := extensions.Remove(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Removed extension %q\n", args[0])
		return nil
	},
}

func init() {
	extensionCmd.AddCommand(extensionAddCmd, extensionRemoveCmd)
	rootCmd.AddCommand(extensionCmd)
}
