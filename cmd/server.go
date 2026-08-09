// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"time"

	"nui/internal/nuiclient"
	"nui/internal/server"

	"github.com/spf13/cobra"
)

var (
	port             int
	agentType        string
	prompt           string
	workingDir       string
	harnessOverride  string
	openBrowser      bool
	noBrowser        bool
	hideInput        bool
	theme            string
	defaultAgentType string
	defaultHarness   string
)

var uiFS func() fs.FS

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the web server",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := server.StartOptions{
			AgentType:        agentType,
			Prompt:           prompt,
			WorkingDir:       workingDir,
			Harness:          harnessOverride,
			Open:             openBrowser && !noBrowser,
			HideInput:        hideInput,
			Theme:            theme,
			DefaultAgentType: defaultAgentType,
			DefaultHarness:   defaultHarness,
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		client := nuiclient.New(fmt.Sprintf("http://127.0.0.1:%d", port))
		if err := client.Health(ctx); err == nil {
			return attachToRunningServer(ctx, port, opts)
		}

		fmt.Printf("Starting web server on port %d...\n", port)
		return server.Start(port, uiFS(), opts)
	},
}

func init() {
	serverCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	serverCmd.Flags().StringVarP(&agentType, "agent-type", "a", "", "Agent id to launch (creates a new session on startup)")
	serverCmd.Flags().StringVarP(&prompt, "prompt", "m", "", "Initial prompt to run in the new session")
	serverCmd.Flags().StringVarP(&workingDir, "working-dir", "w", "", "Working directory for the new session (defaults to current directory)")
	serverCmd.Flags().StringVar(&harnessOverride, "harness", "", "CLI harness override for the launched session (must be allowed by the agent)")
	serverCmd.Flags().BoolVar(&openBrowser, "open", false, "Open the web UI in the system default browser")
	serverCmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open a browser (headless daemon mode)")
	serverCmd.Flags().BoolVar(&hideInput, "hide-input", false, "Hide the chat input (for one-off runs with --prompt)")
	serverCmd.Flags().StringVar(&theme, "theme", "", "UI theme: light or dark (saved to ~/.nui/settings.json)")
	serverCmd.Flags().StringVar(&defaultAgentType, "default-agent", "", "Default ADL agent id for new sessions (saved to ~/.nui/settings.json)")
	serverCmd.Flags().StringVar(&defaultHarness, "default-harness", "", "Default harness for internal agents (e.g. api/anthropic, claude-code; saved to ~/.nui/settings.json)")
	rootCmd.AddCommand(serverCmd)
}

// SetUIFS is called from main to inject the embedded UI FS provider.
func SetUIFS(fn func() fs.FS) {
	uiFS = fn
}

func ensureNuiServer(ctx context.Context, client *nuiclient.Client, spawn bool) error {
	if err := client.Health(ctx); err == nil {
		return nil
	}
	if !spawn {
		return fmt.Errorf("nui server not reachable at %s (start with `nui server` or pass --spawn)", client.BaseURL)
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "server", "--port", "8080", "--no-browser")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("spawn nui server: %w", err)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if err := client.Health(ctx); err == nil {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for nui server at %s", client.BaseURL)
}
