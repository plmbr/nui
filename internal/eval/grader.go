// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package eval

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"loop/internal/loopclient"
	"loop/internal/model"
)

// GradeResult is the outcome of grading agent output.
type GradeResult struct {
	Passed  *bool  // nil when no auto-grade (expect none or missing)
	Message string // failure reason or success note
}

// Grader grades agent output against an eval expectation.
type Grader struct {
	Client      *loopclient.Client
	JudgeAgent  string // agent id for llm judge runs; default claude-code
}

func (g *Grader) judgeAgent() string {
	if strings.TrimSpace(g.JudgeAgent) != "" {
		return strings.TrimSpace(g.JudgeAgent)
	}
	return "claude-code"
}

// Grade evaluates output against ev.Expect.
func (g *Grader) Grade(ctx context.Context, output string, expect *model.ADLEvalExpect) (GradeResult, error) {
	if expect == nil {
		return GradeResult{}, nil
	}
	switch strings.TrimSpace(expect.Type) {
	case "", "none":
		return GradeResult{}, nil
	case "contains":
		ok := strings.Contains(strings.ToLower(output), strings.ToLower(strings.TrimSpace(expect.Value)))
		passed := ok
		msg := "output contains expected value"
		if !ok {
			msg = fmt.Sprintf("expected contains %q", expect.Value)
		}
		return GradeResult{Passed: &passed, Message: msg}, nil
	case "exact":
		ok := strings.TrimSpace(output) == strings.TrimSpace(expect.Value)
		passed := ok
		msg := "exact match"
		if !ok {
			msg = fmt.Sprintf("expected exact %q", expect.Value)
		}
		return GradeResult{Passed: &passed, Message: msg}, nil
	case "regex":
		re, err := regexp.Compile(expect.Value)
		if err != nil {
			return GradeResult{}, fmt.Errorf("invalid regex: %w", err)
		}
		ok := re.MatchString(output)
		passed := ok
		msg := "regex matched"
		if !ok {
			msg = fmt.Sprintf("expected regex %q", expect.Value)
		}
		return GradeResult{Passed: &passed, Message: msg}, nil
	case "llm":
		if g.Client == nil {
			return GradeResult{}, fmt.Errorf("llm judge requires a loop client")
		}
		return g.gradeLLM(ctx, output, expect.Criteria)
	default:
		return GradeResult{}, fmt.Errorf("unknown expect type %q", expect.Type)
	}
}

func (g *Grader) gradeLLM(ctx context.Context, output, criteria string) (GradeResult, error) {
	prompt := fmt.Sprintf(`You are an eval judge. Reply with exactly YES or NO on the first line.

Criteria:
%s

Agent output:
%s

Does the agent output satisfy the criteria?`, strings.TrimSpace(criteria), strings.TrimSpace(output))

	sess, err := g.Client.CreateSession(ctx, loopclient.CreateSessionRequest{
		AgentType: g.judgeAgent(),
	})
	if err != nil {
		return GradeResult{}, fmt.Errorf("llm judge session: %w", err)
	}
	started, err := g.Client.StartRun(ctx, sess.ID, loopclient.StartRunRequest{Message: prompt})
	if err != nil {
		return GradeResult{}, fmt.Errorf("llm judge run: %w", err)
	}
	rec, err := g.Client.WaitRun(ctx, sess.ID, started.RunID, 0)
	if err != nil {
		return GradeResult{}, fmt.Errorf("llm judge wait: %w", err)
	}
	if rec.Status != "completed" {
		return GradeResult{}, fmt.Errorf("llm judge run %s: %s", rec.Status, rec.Error)
	}
	answer := strings.ToUpper(strings.TrimSpace(rec.Output))
	passed := strings.HasPrefix(answer, "YES")
	msg := "llm judge: pass"
	if !passed {
		msg = "llm judge: fail"
	}
	return GradeResult{Passed: &passed, Message: msg}, nil
}
