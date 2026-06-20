// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

type persistentClaudeSession struct {
	mu sync.Mutex

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	binaryPath   string
	model        string
	systemPrompt string
	sandbox      string
	useBwrap     bool
	workingDir   string

	claudeSessionID string
}

func (s *persistentClaudeSession) matches(agent *ClaudeCodeAgent, req RunRequest) bool {
	if s == nil || s.cmd == nil || !processAlive(s.cmd) {
		return false
	}
	model := agent.modelName()
	useBwrap := agent.useBwrap()
	return s.binaryPath == agent.binaryPath() &&
		s.model == model &&
		s.systemPrompt == req.SystemPrompt &&
		s.sandbox == agent.Sandbox &&
		s.useBwrap == useBwrap &&
		s.workingDir == req.WorkingDir
}

func processAlive(cmd *exec.Cmd) bool {
	if cmd == nil || cmd.Process == nil || cmd.ProcessState != nil {
		return false
	}
	return cmd.Process.Signal(syscall.Signal(0)) == nil
}

func (s *persistentClaudeSession) runTurn(ctx context.Context, agent *ClaudeCodeAgent, req RunRequest, events chan<- Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureProcess(ctx, agent, req); err != nil {
		return err
	}

	userMsg := map[string]any{
		"type": "user",
		"message": map[string]any{
			"role":    "user",
			"content": []map[string]string{{"type": "text", "text": req.Message}},
		},
	}
	sessionID := s.claudeSessionID
	if sessionID == "" {
		sessionID = req.SessionID
	}
	if sessionID != "" {
		userMsg["session_id"] = sessionID
	}

	payload, err := json.Marshal(userMsg)
	if err != nil {
		return err
	}
	payload = append(payload, '\n')

	if err := s.writePayload(payload); err != nil {
		if err2 := s.restart(ctx, agent, req); err2 != nil {
			return err2
		}
		if err := s.writePayload(payload); err != nil {
			return fmt.Errorf("write claude stdin after restart: %w", err)
		}
	}

	parser := newClaudeStreamParser()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !s.stdout.Scan() {
			s.stopLocked()
			return fmt.Errorf("claude process ended unexpectedly")
		}

		line := s.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var envelope struct {
			Type      string `json:"type"`
			SessionID string `json:"session_id"`
			IsError   bool   `json:"is_error"`
			Error     any    `json:"error"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			continue
		}

		parser.handleLine(line, events)

		switch envelope.Type {
		case "result":
			if !envelope.IsError && envelope.SessionID != "" {
				s.claudeSessionID = envelope.SessionID
			}
			return nil
		case "error":
			msg := formatClaudeError(envelope.Error)
			events <- Event{Type: EventError, Error: msg}
			return nil
		}
	}
}

func formatClaudeError(err any) string {
	switch v := err.(type) {
	case string:
		if v != "" {
			return v
		}
	case map[string]any:
		if msg, _ := v["message"].(string); msg != "" {
			return msg
		}
		b, _ := json.Marshal(v)
		if len(b) > 0 {
			return string(b)
		}
	}
	return "unknown error"
}

func (s *persistentClaudeSession) writePayload(payload []byte) error {
	if s.stdin == nil {
		return fmt.Errorf("claude stdin unavailable")
	}
	_, err := s.stdin.Write(payload)
	return err
}

func (s *persistentClaudeSession) ensureProcess(ctx context.Context, agent *ClaudeCodeAgent, req RunRequest) error {
	if s.matches(agent, req) {
		return nil
	}
	s.stopLocked()
	return s.start(ctx, agent, req)
}

func (s *persistentClaudeSession) restart(ctx context.Context, agent *ClaudeCodeAgent, req RunRequest) error {
	s.stopLocked()
	return s.start(ctx, agent, req)
}

func (s *persistentClaudeSession) start(ctx context.Context, agent *ClaudeCodeAgent, req RunRequest) error {
	bin := agent.binaryPath()
	model := agent.modelName()
	useBwrap := agent.useBwrap()

	if req.SessionID != "" && s.claudeSessionID == "" {
		s.claudeSessionID = req.SessionID
	}

	args := []string{
		"-p",
		"--input-format", "stream-json",
		"--output-format", "stream-json",
		"--verbose",
		"--dangerously-skip-permissions",
		"--include-partial-messages",
		"--model", model,
	}
	if req.SystemPrompt != "" {
		args = append(args, "--system-prompt", req.SystemPrompt)
	}
	resume := s.claudeSessionID
	if resume == "" {
		resume = req.SessionID
	}
	if resume != "" {
		args = append(args, "--resume", resume)
	}

	var cmd *exec.Cmd
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

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
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
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "[claude stderr] %s\n", scanner.Text())
		}
	}()

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	s.cmd = cmd
	s.stdin = stdin
	s.stdout = sc
	s.binaryPath = bin
	s.model = model
	s.systemPrompt = req.SystemPrompt
	s.sandbox = agent.Sandbox
	s.useBwrap = useBwrap
	s.workingDir = req.WorkingDir
	return nil
}

func (s *persistentClaudeSession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *persistentClaudeSession) stopLocked() {
	if s.stdin != nil {
		s.stdin.Close()
		s.stdin = nil
	}
	if s.cmd != nil && s.cmd.Process != nil && s.cmd.ProcessState == nil {
		s.cmd.Process.Kill()
		_, _ = s.cmd.Process.Wait()
	}
	s.cmd = nil
	s.stdout = nil
}
