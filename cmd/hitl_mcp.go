// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"loop/internal/loopclient"
	"loop/internal/mcpserver"

	"github.com/spf13/cobra"
)

var hitlMCPURL string

var hitlMCPCmd = &cobra.Command{
	Use:   "hitl-mcp",
	Short: "Run the Loop HITL MCP server (stdio) for ask_user and request_approval",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		baseURL := hitlMCPURL
		if baseURL == "" {
			baseURL = loopclient.New("").BaseURL
		}
		return mcpserver.RunHITL(ctx, baseURL)
	},
}

func init() {
	hitlMCPCmd.Flags().StringVar(&hitlMCPURL, "url", "", "Loop server base URL (default LOOP_API_URL, LOOP_URL, or http://127.0.0.1:8080)")
	rootCmd.AddCommand(hitlMCPCmd)
}
