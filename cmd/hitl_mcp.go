// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"nui/internal/nuiclient"
	"nui/internal/mcpserver"

	"github.com/spf13/cobra"
)

var hitlMCPURL string

var hitlMCPCmd = &cobra.Command{
	Use:   "hitl-mcp",
	Short: "Run the nui HITL MCP server (stdio) for ask_user and request_approval",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		baseURL := hitlMCPURL
		if baseURL == "" {
			baseURL = nuiclient.New("").BaseURL
		}
		return mcpserver.RunHITL(ctx, baseURL)
	},
}

func init() {
	hitlMCPCmd.Flags().StringVar(&hitlMCPURL, "url", "", "nui server base URL (default NUI_API_URL, NUI_URL, or http://127.0.0.1:8080)")
	rootCmd.AddCommand(hitlMCPCmd)
}
