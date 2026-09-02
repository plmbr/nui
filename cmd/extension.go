// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"nui/internal/extensions"

	"github.com/spf13/cobra"
)

var extensionCmd = &cobra.Command{
	Use:   "extension",
	Short: "Manage nui extensions",
}

var extensionAddYes bool

var extensionAddCmd = &cobra.Command{
	Use:   "add [url-or-path]",
	Short: "Install an extension from a git URL, directory, or zip file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, err := installWithOverwrite(cmd, extensionAddYes, "extension", "", func(overwrite bool) (string, error) {
			return extensions.Install(args[0], overwrite)
		})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Installed extension %q\n", name)
		return nil
	},
}

var extensionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed extensions",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := extensions.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No extensions installed.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tVERSION\tDISPLAY NAME\tSTATUS\tDESCRIPTION")
		for _, e := range entries {
			status := "enabled"
			if e.Disabled {
				status = "disabled"
			}
			displayName := e.DisplayName
			if displayName == "" {
				displayName = e.Name
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", e.Name, e.Version, displayName, status, e.Description)
		}
		return w.Flush()
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
	extensionAddCmd.Flags().BoolVarP(&extensionAddYes, "yes", "y", false, "overwrite without prompting")
	extensionCmd.AddCommand(extensionAddCmd, extensionListCmd, extensionRemoveCmd)
	rootCmd.AddCommand(extensionCmd)
}
