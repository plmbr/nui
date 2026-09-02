// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"nui/internal/skills"

	"github.com/spf13/cobra"
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "Manage agent skills in the nui catalog",
}

var skillsAddYes bool

var skillsAddCmd = &cobra.Command{
	Use:   "add [url-or-path]",
	Short: "Add a skill to ~/.nui/skills/",
	Long: `Add a skill to the nui catalog. The catalog name defaults to the skill directory name.

Local directory:
  nui skills add ./skills/code-review

GitHub URL (tree or blob link to a skill directory or SKILL.md):
  nui skills add https://github.com/example/repo/tree/main/skills/foo
  nui skills add https://github.com/example/repo/blob/main/skills/foo/SKILL.md

Git remote with explicit path:
  nui skills add --git https://github.com/example/repo.git --path skills/foo

Inline content (name from SKILL.md frontmatter when omitted):
  nui skills add --content "$(cat SKILL.md)"`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name, _ := cmd.Flags().GetString("name")
		gitURL, _ := cmd.Flags().GetString("git")
		repoPath, _ := cmd.Flags().GetString("path")
		version, _ := cmd.Flags().GetString("version")
		content, _ := cmd.Flags().GetString("content")

		switch {
		case content != "":
			added, err := installWithOverwrite(cmd, skillsAddYes, "skill", name, func(overwrite bool) (string, error) {
				return skills.InstallContent(name, content, overwrite)
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Added skill %q\n", added)
		case len(args) == 1:
			arg := args[0]
			if skills.IsGitRemote(arg) {
				if cloneURL, path, ref, ok := skills.ParseGitHubURL(arg); ok {
					if gitURL == "" {
						gitURL = cloneURL
					}
					if repoPath == "" {
						repoPath = path
					}
					if version == "" && ref != "" {
						version = ref
					}
				} else if gitURL == "" {
					gitURL = arg
				}
				if gitURL == "" {
					return fmt.Errorf("git url is required")
				}
				if repoPath == "" {
					return fmt.Errorf("path is required: use a GitHub tree/blob URL or --path")
				}
				added, err := installWithOverwrite(cmd, skillsAddYes, "skill", name, func(overwrite bool) (string, error) {
					return skills.InstallGit(name, gitURL, repoPath, version, overwrite)
				})
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Added git skill %q\n", added)
				return nil
			}
			added, err := installWithOverwrite(cmd, skillsAddYes, "skill", name, func(overwrite bool) (string, error) {
				return skills.InstallLocal(name, arg, overwrite)
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Added local skill %q\n", added)
		case gitURL != "":
			if repoPath == "" {
				return fmt.Errorf("--path is required with --git")
			}
			added, err := installWithOverwrite(cmd, skillsAddYes, "skill", name, func(overwrite bool) (string, error) {
				return skills.InstallGit(name, gitURL, repoPath, version, overwrite)
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Added git skill %q\n", added)
		default:
			return fmt.Errorf("provide a local path or git URL, or use --git or --content")
		}
		return nil
	},
}

var skillsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List skills in the nui catalog",
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
	Short: "Remove a skill from the nui catalog",
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
	skillsAddCmd.Flags().BoolVarP(&skillsAddYes, "yes", "y", false, "overwrite without prompting")
	skillsAddCmd.Flags().String("name", "", "Skill name in the catalog (defaults to skill directory name)")
	skillsAddCmd.Flags().String("git", "", "Git repository URL")
	skillsAddCmd.Flags().String("path", "", "Relative path to skill directory or SKILL.md in repo")
	skillsAddCmd.Flags().String("version", "", "Git tag or commit (optional; inferred from GitHub tree/blob URLs)")
	skillsAddCmd.Flags().String("content", "", "Inline SKILL.md content")

	skillsCmd.AddCommand(skillsAddCmd, skillsListCmd, skillsRemoveCmd)
	rootCmd.AddCommand(skillsCmd)
}
