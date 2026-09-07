// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"nui/internal/mcpserver"

	"github.com/spf13/cobra"
)

var bashMCPCmd = &cobra.Command{
	Use:   "bash-mcp",
	Short: "Run the nui bash MCP server (stdio) for shell commands",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		return mcpserver.RunBash(ctx)
	},
}

func init() {
	rootCmd.AddCommand(bashMCPCmd)
}
