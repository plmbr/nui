// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"os"
	"strings"

	"nui/internal/mcpserver"

	"github.com/spf13/cobra"
)

var orchestratorMCPCmd = &cobra.Command{
	Use:   "orchestrator-mcp",
	Short: "Run the nui orchestrator MCP server (stdio) for launcher routing",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		baseURL := strings.TrimSpace(os.Getenv("NUI_API_URL"))
		if baseURL == "" {
			baseURL = strings.TrimSpace(os.Getenv("NUI_URL"))
		}
		return mcpserver.RunOrchestrator(ctx, baseURL)
	},
}

func init() {
	rootCmd.AddCommand(orchestratorMCPCmd)
}
