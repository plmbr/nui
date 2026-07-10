// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"loop/internal/eval"
	"loop/internal/loopclient"

	"github.com/spf13/cobra"
)

var (
	evalAgentType   string
	evalWorkingDir  string
	evalURL         string
	evalSpawn       bool
	evalJSON        bool
	evalParallel    int
	evalCaseNames   []string
)

var agentEvalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Run agent eval test cases",
}

var agentEvalRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run eval test cases for an agent",
	Long: `Execute eval cases from an agent's ADL definition against a running Loop server.

Examples:
  loop agent eval run -a my-agent
  loop agent eval run -a my-agent --case smoke --case regression
  loop agent eval run -a my-agent -w ./fixtures --json
  loop agent eval run -a my-agent --parallel 2 --spawn`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if strings.TrimSpace(evalAgentType) == "" {
			return fmt.Errorf("--agent-type (-a) is required")
		}
		return nil
	},
	RunE: runAgentEval,
}

func runAgentEval(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	client := loopclient.New(evalURL)
	if err := ensureLoopServer(ctx, client, evalSpawn); err != nil {
		return err
	}

	runner := &eval.Runner{Client: client}
	summary, err := runner.Run(ctx, eval.Options{
		AgentID:     strings.TrimSpace(evalAgentType),
		WorkingDir:  evalWorkingDir,
		FilterNames: evalCaseNames,
		Parallel:    evalParallel,
	})
	if err != nil {
		return err
	}

	if evalJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summary)
	}

	for _, res := range summary.Results {
		label := strings.ToUpper(res.Status)
		detail := res.Message
		if res.Error != "" {
			detail = res.Error
		} else if res.Status == "fail" && detail == "" {
			detail = "assertion failed"
		}
		if detail != "" {
			fmt.Printf("eval %-24s %s  (%s) — %s\n", res.Name, label, res.Duration, detail)
		} else {
			fmt.Printf("eval %-24s %s  (%s)\n", res.Name, label, res.Duration)
		}
	}
	fmt.Printf("---\n%d evals: %d passed, %d failed", len(summary.Results), summary.Passed, summary.Failed)
	if summary.Errors > 0 {
		fmt.Printf(", %d errors", summary.Errors)
	}
	if summary.Skipped > 0 {
		fmt.Printf(", %d skipped", summary.Skipped)
	}
	fmt.Println()

	if summary.Failed > 0 || summary.Errors > 0 {
		return fmt.Errorf("eval run failed")
	}
	return nil
}

func init() {
	agentEvalRunCmd.Flags().StringVarP(&evalAgentType, "agent-type", "a", "", "ADL agent id")
	agentEvalRunCmd.Flags().StringVarP(&evalWorkingDir, "working-dir", "w", "", "Default working directory for eval cases")
	agentEvalRunCmd.Flags().StringVar(&evalURL, "url", "", "Loop server base URL (default LOOP_URL or http://127.0.0.1:8080)")
	agentEvalRunCmd.Flags().BoolVar(&evalSpawn, "spawn", false, "Start loop ui in the background if the server is unreachable")
	agentEvalRunCmd.Flags().BoolVar(&evalJSON, "json", false, "Output machine-readable JSON results")
	agentEvalRunCmd.Flags().IntVar(&evalParallel, "parallel", 1, "Number of eval cases to run concurrently")
	agentEvalRunCmd.Flags().StringArrayVar(&evalCaseNames, "case", nil, "Run only evals with this name (repeatable)")
	agentEvalCmd.AddCommand(agentEvalRunCmd)
}
