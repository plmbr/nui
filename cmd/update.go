// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"

	"nui/internal/appversion"
	"nui/internal/update"

	"github.com/spf13/cobra"
)

var (
	updateCheckOnly bool
	updateYes       bool
	updateForce     bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update nui from GitHub Releases",
	Long: `Check GitHub Releases for a newer nui CLI and optionally install it.

  nui update --check   Report whether an update is available (exit 0=up-to-date, 1=available, 3=error)
  nui update           Check, confirm, download, confirm, then replace this binary
  nui update --yes     Non-interactive (download and install without prompts)
  nui update --force   Reinstall the latest release even when versions match
`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	updateCmd.Flags().BoolVar(&updateCheckOnly, "check", false, "only check for updates")
	updateCmd.Flags().BoolVarP(&updateYes, "yes", "y", false, "download and install without prompting")
	updateCmd.Flags().BoolVar(&updateForce, "force", false, "reinstall even if already up to date")
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	current := appversion.Get()
	if current == "" || current == "dev" {
		current = Version
	}
	mgr := update.NewManager(update.KindCLI, current)

	fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", current)
	fmt.Fprintln(cmd.OutOrStdout(), "Checking for updates…")

	st, err := mgr.Check(cmd.Context(), updateForce)
	if err != nil {
		if updateCheckOnly {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
			os.Exit(3)
		}
		return err
	}

	switch st.State {
	case update.StateUpToDate:
		fmt.Fprintf(cmd.OutOrStdout(), "Already up to date (%s).\n", st.CurrentVersion)
		return nil
	case update.StateAvailable:
		fmt.Fprintf(cmd.OutOrStdout(), "Update available: %s → %s\n", st.CurrentVersion, st.AvailableVersion)
	default:
		return fmt.Errorf("unexpected update state %q", st.State)
	}

	if updateCheckOnly {
		os.Exit(1)
		return nil
	}

	if !updateYes {
		ok, err := confirm(cmd, fmt.Sprintf("Download %s?", st.AssetName))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Downloading…")
	st, err = mgr.Download(cmd.Context())
	if err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Downloaded %s (verified).\n", st.AssetName)

	if !updateYes {
		ok, err := confirm(cmd, "Install and replace this binary?")
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
	}

	st, err = mgr.ApplyCLI(update.TargetSelf)
	if err != nil {
		return err
	}
	exe, _ := update.CurrentExecutable()
	fmt.Fprintf(cmd.OutOrStdout(), "Installed %s to %s\n", st.CurrentVersion, exe)
	fmt.Fprintln(cmd.OutOrStdout(), "Restart any running `nui server` processes to pick up the new version.")
	return nil
}
