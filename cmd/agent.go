// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"loop/internal/agents"
	"loop/internal/loopclient"

	"github.com/spf13/cobra"
)

var agentListURL string

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage and run Loop agents",
}

var agentAddCmd = &cobra.Command{
	Use:   "add [url-or-path]",
	Short: "Install an ADL agent YAML into ~/.loop/agents/",
	Long: `Install an agent definition from a local YAML file or git URL.

Local file:
  loop agent add ./my-agent.yaml
  loop agent add dev/adl/examples/17-auto-scheduled-agent.yaml

GitHub URL (tree or blob link to an agent YAML file):
  loop agent add https://github.com/example/repo/blob/main/agents/watchdog.yaml

Git repository (single yaml at repo root):
  loop agent add https://github.com/example/my-agent.git`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := agents.Install(args[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Installed agent %q\n", id)
		return nil
	},
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available agent types",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		client := loopclient.New(agentListURL)
		if err := ensureLoopServer(ctx, client, false); err != nil {
			return err
		}
		items, err := client.ListAgents(ctx)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No agent types found.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "ID\tLABEL\tHARNESS\tPROMPT\tSOURCE\tAVAILABLE")
		for _, a := range items {
			prompt := a.PromptMode
			if prompt == "" {
				prompt = "user"
			}
			source := a.Source
			if source == "" {
				if a.IsBuiltin {
					source = "builtin"
				} else {
					source = "user"
				}
			}
			available := "yes"
			if !a.Available {
				available = "no"
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				a.ID, a.Label, a.Harness, prompt, source, available)
		}
		return w.Flush()
	},
}

var agentRemoveCmd = &cobra.Command{
	Use:   "remove [id-or-file]",
	Short: "Remove a user-installed agent from ~/.loop/agents/",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := agents.Remove(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Removed agent %q\n", args[0])
		return nil
	},
}

func init() {
	agentListCmd.Flags().StringVar(&agentListURL, "url", "", "Loop server base URL (default LOOP_URL or http://127.0.0.1:8080)")
	agentCmd.AddCommand(NewRunCmd(), NewScheduleCmd(), agentListCmd, agentAddCmd, agentRemoveCmd)
	rootCmd.AddCommand(agentCmd)
}
