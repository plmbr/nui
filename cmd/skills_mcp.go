// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"

	"nui/internal/mcpserver"

	"github.com/spf13/cobra"
)

var skillsMCPCmd = &cobra.Command{
	Use:   "skills-mcp",
	Short: "Run the nui skills MCP server (stdio) for list_skills and load_skill",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}
		return mcpserver.RunSkills(ctx)
	},
}

func init() {
	rootCmd.AddCommand(skillsMCPCmd)
}
