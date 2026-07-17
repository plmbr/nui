// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"nui/internal/nuiclient"

	"github.com/spf13/cobra"
)

var (
	runAgentType  string
	runMessage    string
	runWorkingDir string
	runURL        string
	runWait       bool
	runSpawn      bool
	runSessionID  string
)

// NewRunCmd returns the nui run command.
func NewRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run an agent headlessly against a nui server",
		RunE:  runCommand,
	}
	registerRunFlags(cmd)
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if noWait, _ := cmd.Flags().GetBool("no-wait"); noWait {
			runWait = false
		}
		return nil
	}
	return cmd
}

func runCommand(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		client := nuiclient.New(runURL)
		if err := ensureNuiServer(ctx, client, runSpawn); err != nil {
			return err
		}

		sessionID := strings.TrimSpace(runSessionID)
		if sessionID == "" {
			agentType := strings.TrimSpace(runAgentType)
			if agentType == "" {
				var err error
				agentType, err = client.ResolveDefaultAgentType(ctx)
				if err != nil {
					return err
				}
				fmt.Fprintf(os.Stderr, "Using default agent %s\n", agentType)
			}
			wd := strings.TrimSpace(runWorkingDir)
			if wd == "" {
				cwd, err := os.Getwd()
				if err != nil {
					return err
				}
				wd = cwd
			}
			sess, err := client.CreateSession(ctx, nuiclient.CreateSessionRequest{
				AgentType:  agentType,
				WorkingDir: wd,
			})
			if err != nil {
				return err
			}
			sessionID = sess.ID
			fmt.Fprintf(os.Stderr, "Created session %s\n", sessionID)
		}

		msg := strings.TrimSpace(runMessage)
		started, err := client.StartRun(ctx, sessionID, nuiclient.StartRunRequest{Message: msg})
		if err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Started run %s\n", started.RunID)

		if !runWait {
			fmt.Println(started.RunID)
			return nil
		}

		rec, err := client.StreamRunEvents(ctx, sessionID, started.RunID, "", func(data []byte) {
			var ev struct {
				Type    string `json:"type"`
				Content string `json:"content,omitempty"`
			}
			if json.Unmarshal(data, &ev) == nil && ev.Type == "text" && ev.Content != "" {
				fmt.Print(ev.Content)
			}
		})
		if err != nil {
			return err
		}

		if rec.Output != "" {
			if !strings.HasSuffix(rec.Output, "\n") {
				fmt.Println()
			}
		}
		switch rec.Status {
		case "completed":
			return nil
		case "cancelled":
			return fmt.Errorf("run cancelled")
		default:
			if rec.Error != "" {
				return fmt.Errorf("run failed: %s", rec.Error)
			}
			return fmt.Errorf("run failed")
		}
}

func registerRunFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&runAgentType, "agent-type", "a", "", "ADL agent id for a new session (default: settings defaultAgentType)")
	cmd.Flags().StringVarP(&runMessage, "message", "m", "", "Prompt message (optional for promptMode:auto agents)")
	cmd.Flags().StringVarP(&runWorkingDir, "working-dir", "w", "", "Working directory for a new session")
	cmd.Flags().StringVar(&runURL, "url", "", "nui server base URL (default NUI_URL or http://127.0.0.1:8080)")
	cmd.Flags().StringVar(&runSessionID, "session-id", "", "Existing session id (skips session create)")
	cmd.Flags().BoolVar(&runWait, "wait", true, "Wait for the run to finish and stream text to stdout")
	cmd.Flags().Bool("no-wait", false, "Return immediately after starting the run (same as --wait=false)")
	cmd.Flags().BoolVar(&runSpawn, "spawn", false, "Start nui ui in the background if the server is unreachable")
}

func init() {
	rootCmd.AddCommand(NewRunCmd())
}
