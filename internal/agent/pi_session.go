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
	"strings"
	"sync"
)

type persistentPiSession struct {
	mu sync.Mutex

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	stderrLines []string

	binaryPath   string
	model        string
	systemPrompt string
	workingDir   string

	sessionID string
}

func (s *persistentPiSession) currentSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionID
}

func (s *persistentPiSession) runTurn(ctx context.Context, agent *PiAgent, req RunRequest, events chan<- Event) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resume := req.SessionID
	if resume == "" {
		resume = s.sessionID
	}

	if err := s.ensureProcess(ctx, agent, req, resume); err != nil {
		return "", err
	}

	producedOutput, err := s.promptTurn(ctx, req.Message, events)
	if err != nil {
		return "", err
	}

	if !producedOutput && resume != "" && s.hasSessionNotFound() {
		fmt.Fprintln(os.Stderr, "[pi] session not found, retrying without session")
		s.stopLocked()
		s.sessionID = ""
		if err := s.ensureProcess(ctx, agent, req, ""); err != nil {
			return "", err
		}
		if _, err := s.promptTurn(ctx, req.Message, events); err != nil {
			return "", err
		}
	}

	s.refreshSessionID()
	return s.sessionID, nil
}

func (s *persistentPiSession) hasSessionNotFound() bool {
	for _, line := range s.stderrLines {
		if strings.Contains(line, "No session found") {
			return true
		}
	}
	return false
}

func (s *persistentPiSession) promptTurn(ctx context.Context, message string, events chan<- Event) (bool, error) {
	if s.stdin == nil || s.stdout == nil {
		return false, fmt.Errorf("pi stdin unavailable")
	}

	payload, err := json.Marshal(map[string]any{"type": "prompt", "message": message})
	if err != nil {
		return false, err
	}
	payload = append(payload, '\n')

	if _, err := s.stdin.Write(payload); err != nil {
		s.stopLocked()
		return false, fmt.Errorf("pi process ended unexpectedly")
	}

	parser := newPiStreamParser()
	producedOutput := false
	for {
		if err := ctx.Err(); err != nil {
			return producedOutput, err
		}
		if !s.stdout.Scan() {
			s.stopLocked()
			return producedOutput, fmt.Errorf("pi process ended unexpectedly")
		}

		line := s.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var obj struct {
			Type    string `json:"type"`
			Command string `json:"command"`
			Success bool   `json:"success"`
			Error   any    `json:"error"`
		}
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}

		if obj.Type == "response" {
			if obj.Command == "prompt" && !obj.Success {
				msg := formatPiError(obj.Error)
				events <- Event{Type: EventError, Error: msg}
				return producedOutput, nil
			}
			continue
		}
		if obj.Type == "extension_ui_request" || obj.Type == "extension_ui_response" {
			continue
		}

		parser.handleLine(line, events)
		if obj.Type != "error" {
			producedOutput = true
		}
		if obj.Type == "turn_end" {
			return producedOutput, nil
		}
	}
}

func formatPiError(err any) string {
	switch v := err.(type) {
	case string:
		if v != "" {
			return v
		}
	default:
		if v != nil {
			b, _ := json.Marshal(v)
			if len(b) > 0 {
				return string(b)
			}
		}
	}
	return "prompt rejected"
}

func (s *persistentPiSession) refreshSessionID() {
	if s.stdin == nil || s.stdout == nil {
		return
	}
	payload := []byte(`{"type":"get_state"}` + "\n")
	if _, err := s.stdin.Write(payload); err != nil {
		return
	}
	for {
		if !s.stdout.Scan() {
			return
		}
		line := s.stdout.Bytes()
		var obj struct {
			Type    string `json:"type"`
			Command string `json:"command"`
			Success bool   `json:"success"`
			Data    struct {
				SessionID string `json:"sessionId"`
			} `json:"data"`
		}
		if err := json.Unmarshal(line, &obj); err != nil {
			continue
		}
		if obj.Type != "response" || obj.Command != "get_state" {
			continue
		}
		if obj.Success && obj.Data.SessionID != "" {
			s.sessionID = obj.Data.SessionID
		}
		return
	}
}

func (s *persistentPiSession) ensureProcess(ctx context.Context, agent *PiAgent, req RunRequest, resumeSessionID string) error {
	wd := req.WorkingDir
	if wd == "" {
		if cwd, err := os.Getwd(); err == nil {
			wd = cwd
		}
	}
	if s.cmd != nil && processAlive(s.cmd) &&
		s.workingDir == wd &&
		s.model == req.Model &&
		s.systemPrompt == req.SystemPrompt {
		return nil
	}

	s.stopLocked()
	if resumeSessionID != "" && s.sessionID == "" {
		s.sessionID = resumeSessionID
	}

	args := []string{"--mode", "rpc"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.SystemPrompt != "" {
		args = append(args, "--system-prompt", req.SystemPrompt)
	}
	resume := s.sessionID
	if resume == "" {
		resume = resumeSessionID
	}
	if resume != "" {
		args = append(args, "--session", resume)
	}

	cmd := exec.CommandContext(ctx, agent.binaryPath(), args...)
	if wd != "" {
		cmd.Dir = wd
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
		fmt.Fprintf(os.Stderr, "[pi] start error: %v\n", err)
		return err
	}

	s.stderrLines = nil
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := scanner.Text()
			s.mu.Lock()
			s.stderrLines = append(s.stderrLines, text)
			s.mu.Unlock()
			fmt.Fprintf(os.Stderr, "[pi stderr] %s\n", text)
		}
	}()

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	s.cmd = cmd
	s.stdin = stdin
	s.stdout = sc
	s.workingDir = wd
	s.model = req.Model
	s.systemPrompt = req.SystemPrompt
	s.binaryPath = agent.binaryPath()
	return nil
}

func (s *persistentPiSession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *persistentPiSession) stopLocked() {
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
	s.workingDir = ""
}
