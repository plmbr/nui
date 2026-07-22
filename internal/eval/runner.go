// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package eval

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"nui/internal/agents"
	"nui/internal/nuiclient"
	"nui/internal/model"
)

// Result is the outcome of running one eval case.
type Result struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // pass | fail | error | skip
	Output   string `json:"output,omitempty"`
	Passed   *bool  `json:"passed,omitempty"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
	Duration string `json:"duration"`
}

// Summary aggregates eval run results.
type Summary struct {
	AgentID string   `json:"agentId"`
	Results []Result `json:"results"`
	Passed  int      `json:"passed"`
	Failed  int      `json:"failed"`
	Errors  int      `json:"errors"`
	Skipped int      `json:"skipped"`
}

// Options configures an eval run.
type Options struct {
	AgentID     string
	WorkingDir  string
	FilterNames []string // empty = all enabled evals
	Parallel    int
	JudgeAgent  string
}

// Runner executes eval cases against a nui server.
type Runner struct {
	Client *nuiclient.Client
}

// Run executes eval cases for the given agent.
func (r *Runner) Run(ctx context.Context, opts Options) (Summary, error) {
	def, ok := agents.LookupDefinition(opts.AgentID)
	if !ok {
		return Summary{}, fmt.Errorf("agent %q not found", opts.AgentID)
	}
	if len(def.Evals) == 0 {
		return Summary{}, fmt.Errorf("agent %q has no evals defined", opts.AgentID)
	}

	agentID := model.ADLAgentID(def)
	cases := filterEvals(def.Evals, opts.FilterNames)
	if len(cases) == 0 {
		return Summary{}, fmt.Errorf("no matching eval cases")
	}

	parallel := opts.Parallel
	if parallel < 1 {
		parallel = 1
	}

	grader := &Grader{Client: r.Client, JudgeAgent: opts.JudgeAgent}
	results := make([]Result, len(cases))
	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup
	var mu sync.Mutex
	summary := Summary{AgentID: agentID}

	for i, ev := range cases {
		wg.Add(1)
		go func(idx int, evalCase model.ADLEval) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := r.runCase(ctx, def, evalCase, agentID, opts.WorkingDir, grader)
			mu.Lock()
			results[idx] = res
			switch res.Status {
			case "pass":
				summary.Passed++
			case "fail":
				summary.Failed++
			case "error":
				summary.Errors++
			case "skip":
				summary.Skipped++
			}
			mu.Unlock()
		}(i, ev)
	}
	wg.Wait()
	summary.Results = results
	return summary, nil
}

func filterEvals(evals []model.ADLEval, names []string) []model.ADLEval {
	if len(names) == 0 {
		out := make([]model.ADLEval, 0, len(evals))
		for _, ev := range evals {
			if !ev.Disabled {
				out = append(out, ev)
			}
		}
		return out
	}
	want := map[string]bool{}
	for _, n := range names {
		want[strings.TrimSpace(n)] = true
	}
	var out []model.ADLEval
	for _, ev := range evals {
		if want[ev.Name] && !ev.Disabled {
			out = append(out, ev)
		}
	}
	return out
}

func (r *Runner) runCase(
	ctx context.Context,
	def model.ADLDefinition,
	ev model.ADLEval,
	agentID string,
	defaultWorkingDir string,
	grader *Grader,
) Result {
	start := time.Now()
	result := Result{Name: ev.Name}

	timeout := model.EffectiveEvalTimeout(def, ev)
	caseCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	wd, err := resolveWorkingDir(defaultWorkingDir, ev.WorkingDir)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(start).String()
		return result
	}

	sess, err := r.Client.CreateSession(caseCtx, nuiclient.CreateSessionRequest{
		AgentType:  agentID,
		WorkingDir: wd,
	})
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(start).String()
		return result
	}
	defer func() {
		_ = r.Client.DeleteSession(context.Background(), sess.ID)
	}()

	output, runErr := r.executeEval(caseCtx, sess.ID, ev)
	if runErr != nil {
		result.Status = "error"
		result.Error = runErr.Error()
		result.Duration = time.Since(start).String()
		return result
	}
	result.Output = output

	grade, err := grader.Grade(caseCtx, output, ev.Expect)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		result.Duration = time.Since(start).String()
		return result
	}

	if grade.Passed == nil {
		result.Status = "pass"
		result.Message = "informational run (no grader)"
		result.Duration = time.Since(start).String()
		return result
	}

	result.Passed = grade.Passed
	result.Message = grade.Message
	if *grade.Passed {
		result.Status = "pass"
	} else {
		result.Status = "fail"
	}
	result.Duration = time.Since(start).String()
	return result
}

func (r *Runner) executeEval(ctx context.Context, sessionID string, ev model.ADLEval) (string, error) {
	if strings.TrimSpace(ev.Input) != "" {
		return r.runOnce(ctx, sessionID, ev.Input)
	}
	return r.runConversation(ctx, sessionID, ev.Messages)
}

func (r *Runner) runConversation(ctx context.Context, sessionID string, messages []model.ADLEvalMessage) (string, error) {
	if len(messages) == 0 {
		return "", fmt.Errorf("eval has no messages")
	}
	lastIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if strings.TrimSpace(messages[i].Role) == "user" {
			lastIdx = i
			break
		}
	}
	if lastIdx < 0 {
		return "", fmt.Errorf("eval messages must end with a user turn")
	}

	var lastOutput string
	for i, msg := range messages {
		if strings.TrimSpace(msg.Role) != "user" {
			continue
		}
		content := strings.TrimSpace(msg.Content)
		if content == "" {
			continue
		}
		out, err := r.runOnce(ctx, sessionID, content)
		if err != nil {
			return "", err
		}
		if i == lastIdx {
			lastOutput = out
		}
	}
	return lastOutput, nil
}

func (r *Runner) runOnce(ctx context.Context, sessionID, message string) (string, error) {
	started, err := r.Client.StartRun(ctx, sessionID, nuiclient.StartRunRequest{
		Message: strings.TrimSpace(message),
	})
	if err != nil {
		return "", err
	}
	rec, err := r.waitRunTerminal(ctx, sessionID, started.RunID)
	if err != nil {
		return "", err
	}
	if rec.Status == "awaiting_user" {
		return rec.Output, fmt.Errorf("run paused for HITL (awaiting_user); use hitl.mode: off for unattended evals")
	}
	if rec.Status != "completed" {
		if rec.Error != "" {
			return rec.Output, fmt.Errorf("run %s: %s", rec.Status, rec.Error)
		}
		return rec.Output, fmt.Errorf("run %s", rec.Status)
	}
	return rec.Output, nil
}

func (r *Runner) waitRunTerminal(ctx context.Context, sessionID, runID string) (nuiclient.RunRecord, error) {
	for {
		rec, err := r.Client.GetRun(ctx, sessionID, runID)
		if err != nil {
			return nuiclient.RunRecord{}, err
		}
		switch rec.Status {
		case "completed", "failed", "cancelled", "awaiting_user":
			return rec, nil
		}
		select {
		case <-ctx.Done():
			return rec, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func resolveWorkingDir(defaultDir, override string) (string, error) {
	wd := strings.TrimSpace(override)
	if wd == "" {
		wd = strings.TrimSpace(defaultDir)
	}
	if wd == "" {
		return os.Getwd()
	}
	if !filepath.IsAbs(wd) {
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		wd = filepath.Join(cwd, wd)
	}
	return filepath.Clean(wd), nil
}
