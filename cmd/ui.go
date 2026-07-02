// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"fmt"
	"io/fs"

	"loop/internal/loopclient"
	"loop/internal/server"

	"github.com/spf13/cobra"
)

var (
	port             int
	agentType        string
	prompt           string
	workingDir       string
	openBrowser      bool
	noBrowser        bool
	hideInput        bool
	theme            string
	defaultAgentType string
)

var uiFS func() fs.FS

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the web UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := server.StartOptions{
			AgentType:        agentType,
			Prompt:           prompt,
			WorkingDir:       workingDir,
			Open:             openBrowser && !noBrowser,
			HideInput:        hideInput,
			Theme:            theme,
			DefaultAgentType: defaultAgentType,
		}

		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		client := loopclient.New(fmt.Sprintf("http://127.0.0.1:%d", port))
		if err := client.Health(ctx); err == nil {
			return attachToRunningServer(ctx, port, opts)
		}

		fmt.Printf("Starting web server on port %d...\n", port)
		return server.Start(port, uiFS(), opts)
	},
}

func init() {
	uiCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	uiCmd.Flags().StringVarP(&agentType, "agent-type", "a", "", "Agent id to launch (creates a new session on startup)")
	uiCmd.Flags().StringVarP(&prompt, "prompt", "m", "", "Initial prompt to run in the new session")
	uiCmd.Flags().StringVarP(&workingDir, "working-dir", "w", "", "Working directory for the new session (defaults to current directory)")
	uiCmd.Flags().BoolVar(&openBrowser, "open", false, "Open the web UI in the system default browser")
	uiCmd.Flags().BoolVar(&noBrowser, "no-browser", false, "Do not open a browser (headless daemon mode)")
	uiCmd.Flags().BoolVar(&hideInput, "hide-input", false, "Hide the chat input (for one-off runs with --prompt)")
	uiCmd.Flags().StringVar(&theme, "theme", "", "UI theme: light or dark (saved to ~/.loop/settings.json)")
	uiCmd.Flags().StringVar(&defaultAgentType, "default-agent", "", "Default agent type for new sessions (ADL id or name; saved to ~/.loop/settings.json)")
	rootCmd.AddCommand(uiCmd)
}

// SetUIFS is called from main to inject the embedded UI FS provider.
func SetUIFS(fn func() fs.FS) {
	uiFS = fn
}
