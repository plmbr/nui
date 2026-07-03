// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"loop/internal/loopclient"

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

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run an agent headlessly against a Loop server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		client := loopclient.New(runURL)
		if err := ensureLoopServer(ctx, client, runSpawn); err != nil {
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
			sess, err := client.CreateSession(ctx, loopclient.CreateSessionRequest{
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
		started, err := client.StartRun(ctx, sessionID, loopclient.StartRunRequest{Message: msg})
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
	},
}

func ensureLoopServer(ctx context.Context, client *loopclient.Client, spawn bool) error {
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
	cmd := exec.Command(exe, "ui", "--port", "8080", "--no-browser")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
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
	runCmd.Flags().StringVarP(&runAgentType, "agent-type", "a", "", "ADL agent id for a new session (default: settings defaultAgentType)")
	runCmd.Flags().StringVarP(&runMessage, "message", "m", "", "Prompt message (optional for promptMode:auto agents)")
	runCmd.Flags().StringVarP(&runWorkingDir, "working-dir", "w", "", "Working directory for a new session")
	runCmd.Flags().StringVar(&runURL, "url", "", "Loop server base URL (default LOOP_URL or http://127.0.0.1:8080)")
	runCmd.Flags().StringVar(&runSessionID, "session-id", "", "Existing session id (skips session create)")
	runCmd.Flags().BoolVar(&runWait, "wait", true, "Wait for the run to finish and stream text to stdout")
	runCmd.Flags().Bool("no-wait", false, "Return immediately after starting the run (same as --wait=false)")
	runCmd.Flags().BoolVar(&runSpawn, "spawn", false, "Start loop ui in the background if the server is unreachable")
	rootCmd.AddCommand(runCmd)
	runCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if noWait, _ := cmd.Flags().GetBool("no-wait"); noWait {
			runWait = false
		}
		return nil
	}
}
