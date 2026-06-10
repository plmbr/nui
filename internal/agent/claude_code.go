// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"os/exec"
)

type ClaudeCodeAgent struct {
	// BinaryPath overrides the claude binary location; defaults to "claude" on PATH.
	BinaryPath string
}

func (a *ClaudeCodeAgent) Name() string { return "claude-code" }

func (a *ClaudeCodeAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	bin := a.BinaryPath
	if bin == "" {
		bin = "claude"
	}

	args := []string{
		"-p", req.Message,
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
		"--include-partial-messages",
	}
	if req.SessionID != "" {
		args = append(args, "--resume", req.SessionID)
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
		return err
	}

	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
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

	return cmd.Wait()
}
