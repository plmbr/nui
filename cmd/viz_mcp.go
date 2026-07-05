// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"loop/internal/mcpserver"

	"github.com/spf13/cobra"
)

var vizMCPCmd = &cobra.Command{
	Use:   "viz-mcp",
	Short: "Run the Loop visualization MCP server (stdio) for show_visualization",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		return mcpserver.RunViz(ctx)
	},
}

func init() {
	rootCmd.AddCommand(vizMCPCmd)
}
