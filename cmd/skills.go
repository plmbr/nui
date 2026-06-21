// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"loop/internal/skills"

	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage agent skills in the Loop catalog",
}

var skillsInstallCmd = &cobra.Command{
	Use:   "install [local-path]",
	Short: "Install a skill into ~/.loop/skills/",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		gitURL, _ := cmd.Flags().GetString("git")
		repoPath, _ := cmd.Flags().GetString("path")
		version, _ := cmd.Flags().GetString("version")
		content, _ := cmd.Flags().GetString("content")

		switch {
		case gitURL != "":
			if name == "" {
				return fmt.Errorf("--name is required with --git")
			}
			if err := skills.InstallGit(name, gitURL, repoPath, version); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Installed git skill %q\n", name)
		case content != "":
			if name == "" {
				return fmt.Errorf("--name is required with --content")
			}
			if err := skills.InstallContent(name, content); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Installed content skill %q\n", name)
		case len(args) == 1:
			if name == "" {
				return fmt.Errorf("--name is required when installing from a local path")
			}
			if err := skills.InstallLocal(name, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Installed local skill %q\n", name)
		default:
			return fmt.Errorf("provide a local path, or use --git or --content")
		}
		return nil
	},
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List skills in the Loop catalog",
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := skills.List()
		if err != nil {
			return err
		}
		if len(entries) == 0 {
			fmt.Println("No skills installed.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSOURCE\tDETAILS")
		for _, e := range entries {
			details := e.Path
			if e.Git != "" {
				details = e.Git
				if e.Path != "" {
					details += " (" + e.Path + ")"
				}
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", e.Name, e.Source, details)
		}
		return w.Flush()
	},
}

var skillsRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove a skill from the Loop catalog",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := skills.Remove(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Removed skill %q\n", args[0])
		return nil
	},
}

func init() {
	skillsInstallCmd.Flags().String("name", "", "Skill name in the catalog")
	skillsInstallCmd.Flags().String("git", "", "Git repository URL")
	skillsInstallCmd.Flags().String("path", "", "Relative path to skill directory in repo (required with --git)")
	skillsInstallCmd.Flags().String("version", "", "Git tag or commit (optional)")
	skillsInstallCmd.Flags().String("content", "", "Inline SKILL.md content")

	skillsCmd.AddCommand(skillsInstallCmd, skillsListCmd, skillsRemoveCmd)
	rootCmd.AddCommand(skillsCmd)
}
