// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"loop/internal/loopclient"
	"loop/internal/mcpserver"

	"github.com/spf13/cobra"
)

var (
	mcpURL   string
	mcpSpawn bool
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run Loop as an MCP server (stdio) for agent discovery and runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		client := loopclient.New(mcpURL)
		if err := ensureLoopServer(ctx, client, mcpSpawn); err != nil {
			return err
		}

		return mcpserver.Run(ctx, client.BaseURL)
	},
}

func init() {
	mcpCmd.Flags().StringVar(&mcpURL, "url", "", "Loop server base URL (default LOOP_URL or http://127.0.0.1:8080)")
	mcpCmd.Flags().BoolVar(&mcpSpawn, "spawn", false, "Start loop ui in the background if the server is unreachable")
	rootCmd.AddCommand(mcpCmd)
}
