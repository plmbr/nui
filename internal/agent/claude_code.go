// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
)

type ClaudeCodeAgent struct {
	// BinaryPath overrides the claude binary location; defaults to "claude" on PATH.
	BinaryPath string
	// Model overrides the default model (e.g. "claude-opus-4-8").
	Model string
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

	cmd := exec.CommandContext(ctx, bin, args...)
	if req.WorkingDir != "" {
		cmd.Dir = req.WorkingDir
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

	scanner := bufio.NewScanner(stdout)
	// 4 MB buffer — claude lines can be large (init message with tool list)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var envelope struct {
			Type      string          `json:"type"`
			Event     json.RawMessage `json:"event"`
			SessionID string          `json:"session_id"`
			IsError   bool            `json:"is_error"`
			ErrMsg    string          `json:"error"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			fmt.Fprintf(os.Stderr, "[claude stdout] %s\n", line)
			continue
		}

		switch envelope.Type {
		case "stream_event":
			var ev struct {
				Type  string `json:"type"`
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal(envelope.Event, &ev); err != nil {
				continue
			}
			if ev.Type == "content_block_delta" && ev.Delta.Type == "text_delta" && ev.Delta.Text != "" {
				events <- Event{Type: EventText, Content: ev.Delta.Text}
			}

		case "result":
			if envelope.IsError {
				events <- Event{Type: EventError, Error: envelope.ErrMsg}
			} else {
				events <- Event{Type: EventDone, SessionID: envelope.SessionID}
			}
		}
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
