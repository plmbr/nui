// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"fmt"
	"io/fs"

	"loop/internal/server"

	"github.com/spf13/cobra"
)

var (
	port       int
	agentType  string
	prompt     string
	workingDir string
)

var uiFS func() fs.FS

var uiCmd = &cobra.Command{
	Use:   "ui",
	Short: "Start the web UI server",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("Starting web server on port %d...\n", port)
		return server.Start(port, uiFS(), server.StartOptions{
			AgentType:  agentType,
			Prompt:     prompt,
			WorkingDir: workingDir,
		})
	},
}

func init() {
	uiCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port to listen on")
	uiCmd.Flags().StringVarP(&agentType, "agent-type", "a", "", "Agent type to launch (creates a new session on startup)")
	uiCmd.Flags().StringVarP(&prompt, "prompt", "m", "", "Initial prompt to run in the new session")
	uiCmd.Flags().StringVarP(&workingDir, "working-dir", "w", "", "Working directory for the new session (defaults to current directory)")
	rootCmd.AddCommand(uiCmd)
}

// SetUIFS is called from main to inject the embedded UI FS provider.
func SetUIFS(fn func() fs.FS) {
	uiFS = fn
}
