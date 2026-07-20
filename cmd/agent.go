// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"nui/internal/agents"
	"nui/internal/nuiclient"

	"github.com/spf13/cobra"
)

var agentListURL string

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Manage and run nui agents",
}

var agentAddCmd = &cobra.Command{
	Use:   "add [url-or-path]",
	Short: "Install an ADL agent YAML into ~/.nui/agents/",
	Long: `Install an agent definition from a local YAML file or git URL.

Local file:
  nui agent add ./my-agent.yaml
  nui agent add dev/adl/examples/simple-cli-agent.yaml

GitHub URL (tree or blob link to an agent YAML file):
  nui agent add https://github.com/example/repo/blob/main/agents/watchdog.yaml

Git repository (single yaml at repo root):
  nui agent add https://github.com/example/my-agent.git`,
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
		client := nuiclient.New(agentListURL)
		if err := ensureNuiServer(ctx, client, false); err != nil {
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
	Short: "Remove a user-installed agent from ~/.nui/agents/",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := agents.Remove(args[0]); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Removed agent %q\n", args[0])
		return nil
	},
}

var agentDeployCmd = &cobra.Command{
	Use:   "deploy [deployer-id] [agent-id]",
	Short: "Deploy a user-installed agent via an extension agent deployer",
	Long: `Deploy an ADL agent using an extension agentDeployer.

Deployer ids use the ext:<extension>/<name> convention, for example:
  nui agent deploy ext:docker-deployer/docker my-agent

Registry, image tags, and platform details are configured inside the deployer extension.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := agents.Deploy(args[0], args[1])
		if err != nil {
			return err
		}
		if result.Message != "" {
			fmt.Println(result.Message)
		}
		if result.DeploymentID != "" {
			fmt.Fprintf(os.Stderr, "deploymentId: %s\n", result.DeploymentID)
		}
		if result.Endpoint != nil {
			ep := result.Endpoint
			switch {
			case ep.URL != "":
				fmt.Fprintf(os.Stderr, "endpoint: %s\n", ep.URL)
			case ep.Host != "" && ep.Port > 0:
				fmt.Fprintf(os.Stderr, "endpoint: %s:%d\n", ep.Host, ep.Port)
			}
		}
		return nil
	},
}

var agentDeployersCmd = &cobra.Command{
	Use:   "deployers",
	Short: "List installed extension agent deployers",
	RunE: func(cmd *cobra.Command, args []string) error {
		items, err := agents.ListDeployers()
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No agent deployers found.")
			return nil
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "DEPLOYER ID\tEXTENSION\tNAME\tDESCRIPTION")
		for _, d := range items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", d.ID, d.Extension, d.Name, d.Description)
		}
		return w.Flush()
	},
}

func init() {
	agentListCmd.Flags().StringVar(&agentListURL, "url", "", "nui server base URL (default NUI_URL or http://127.0.0.1:8080)")
	agentCmd.AddCommand(NewRunCmd(), NewScheduleCmd(), agentListCmd, agentAddCmd, agentRemoveCmd, agentDeployCmd, agentDeployersCmd, agentEvalCmd)
	rootCmd.AddCommand(agentCmd)
}
