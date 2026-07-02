// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

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
		if err := ensureMCPServer(ctx, client, mcpSpawn); err != nil {
			return err
		}

		return mcpserver.Run(ctx, client.BaseURL)
	},
}

func ensureMCPServer(ctx context.Context, client *loopclient.Client, spawn bool) error {
	if err := client.Health(ctx); err == nil {
		return nil
	}
	if !spawn {
		return fmt.Errorf("loop server not reachable at %s (start with `loop ui` or pass --spawn)", client.BaseURL)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	child := exec.Command(exe, "ui", "--port", "8080", "--no-browser")
	child.Stdout = os.Stderr
	child.Stderr = os.Stderr
	if err := child.Start(); err != nil {
		return fmt.Errorf("spawn loop ui: %w", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Health(ctx); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for loop server at %s", client.BaseURL)
}

func init() {
	mcpCmd.Flags().StringVar(&mcpURL, "url", "", "Loop server base URL (default LOOP_URL or http://127.0.0.1:8080)")
	mcpCmd.Flags().BoolVar(&mcpSpawn, "spawn", false, "Start loop ui in the background if the server is unreachable")
	rootCmd.AddCommand(mcpCmd)
}
