// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
)

type ClaudeCodeAgent struct {
	// BinaryPath overrides the claude binary location; defaults to "claude" on PATH.
	BinaryPath string
	// Model overrides the default model (e.g. "claude-opus-4-8").
	Model string
	// Sandbox controls sandboxing: "none" disables bwrap, "bubblewrap" forces it,
	// "" auto-detects (uses bwrap if available — legacy behaviour).
	Sandbox string
}

func (a *ClaudeCodeAgent) Name() string { return "claude-code" }

func (a *ClaudeCodeAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	bin := a.BinaryPath
	if bin == "" {
		bin = "claude"
	}

	model := a.Model
	if model == "" {
		model = "claude-sonnet-4-6"
	}

	args := []string{
		"-p", req.Message,
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
		"--include-partial-messages",
		"--model", model,
	}
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
	}
	if req.SystemPrompt != "" {
		args = append(args, "--system-prompt", req.SystemPrompt)
	}

	var cmd *exec.Cmd
	useBwrap := false
	switch a.Sandbox {
	case "bubblewrap":
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
		useBwrap = true
	case "none":
		useBwrap = false
	default:
		// Legacy auto-detect: use bwrap when available on the host.
		useBwrap = GetBwrapStatus().Available
	}

	if useBwrap {
		bwrap := GetBwrapStatus()
		wrappedBin, wrappedArgs := WrapWithBwrap(bwrap.Path, bin, args, req.WorkingDir)
		cmd = exec.CommandContext(ctx, wrappedBin, wrappedArgs...)
	} else {
		cmd = exec.CommandContext(ctx, bin, args...)
		if req.WorkingDir != "" {
			cmd.Dir = req.WorkingDir
		}
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[claude] start error: %v\n", err)
		return err
	}

	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			fmt.Fprintf(os.Stderr, "[claude stderr] %s\n", s.Text())
		}
	}()

	parser := newClaudeStreamParser()
	scanner := bufio.NewScanner(stdout)
	// 4 MB buffer — claude lines can be large (init message with tool list)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		parser.handleLine(scanner.Bytes(), events)
	}

	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "[claude] exit code %d: %v\n", exitErr.ExitCode(), exitErr)
		} else {
			fmt.Fprintf(os.Stderr, "[claude] exit error: %v\n", err)
		}
	}
	return err
}
