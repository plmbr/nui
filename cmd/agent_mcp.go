// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"nui/internal/mcpserver"

	"github.com/spf13/cobra"
)

var agentMCPCmd = &cobra.Command{
	Use:   "agent-mcp",
	Short: "Run the nui agent MCP server (stdio) for save_agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		return mcpserver.RunAgent(ctx)
	},
}

func init() {
	rootCmd.AddCommand(agentMCPCmd)
}
