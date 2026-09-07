// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"nui/internal/mcpserver"

	"github.com/spf13/cobra"
)

var fsMCPCmd = &cobra.Command{
	Use:   "fs-mcp",
	Short: "Run the nui filesystem MCP server (stdio) for read, write, edit, glob",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		return mcpserver.RunFS(ctx)
	},
}

func init() {
	rootCmd.AddCommand(fsMCPCmd)
}
