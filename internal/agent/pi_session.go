// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
)

var errPiProcessEnded = errors.New("pi process ended unexpectedly")

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
	sandbox      string
	useBwrap     bool
	devcontainerWorkspace string
	configDir    string

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

	if err := s.ensureProcess(agent, req, resume); err != nil {
		return "", err
	}

	producedOutput, err := s.promptTurn(ctx, req.Message, events)
	if err != nil {
		if !errors.Is(err, errPiProcessEnded) {
			return "", err
		}
		fmt.Fprintln(os.Stderr, "[pi] process ended during turn, restarting")
		s.stopLocked()
		if resume != "" && s.sessionID == "" {
			s.sessionID = resume
		}
		if err := s.ensureProcess(agent, req, resume); err != nil {
			return "", err
		}
		producedOutput, err = s.promptTurn(ctx, req.Message, events)
		if err != nil {
			return "", err
		}
	}

	if !producedOutput && resume != "" && s.hasSessionNotFound() {
		fmt.Fprintln(os.Stderr, "[pi] session not found, retrying without session")
		s.stopLocked()
		s.sessionID = ""
		if err := s.ensureProcess(agent, req, ""); err != nil {
			return "", err
		}
		if _, err := s.promptTurn(ctx, req.Message, events); err != nil {
			return "", err
		}
	}

	s.refreshSessionID()
	if s.sessionID == "" {
		s.sessionID = resume
	}
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
		return false, errPiProcessEnded
	}

	payload, err := json.Marshal(map[string]any{"type": "prompt", "message": message})
	if err != nil {
		return false, err
	}
	payload = append(payload, '\n')

	if err := s.writeStdin(payload); err != nil {
		s.stopLocked()
		return false, errPiProcessEnded
	}

	parser := newPiStreamParser()
	producedOutput := false
	for {
		if err := ctx.Err(); err != nil {
			return producedOutput, err
		}
		if !s.stdout.Scan() {
			s.stopLocked()
			return producedOutput, errPiProcessEnded
		}

		line := s.stdout.Bytes()
		if len(line) == 0 {
			continue
		}

		var obj struct {
			Type    string `json:"type"`
			ID      string `json:"id"`
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
		if obj.Type == "session" && obj.ID != "" {
			s.sessionID = obj.ID
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
	if err := s.writeStdin(payload); err != nil {
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

func (s *persistentPiSession) ensureProcess(agent *PiAgent, req RunRequest, resumeSessionID string) error {
	wd := req.WorkingDir
	if wd == "" {
		if cwd, err := os.Getwd(); err == nil {
			wd = cwd
		}
	}
	if s.cmd != nil && processAlive(s.cmd) && s.stdin != nil && s.stdout != nil &&
		s.workingDir == wd &&
		s.model == req.Model &&
		s.systemPrompt == req.SystemPrompt &&
		s.sandbox == agent.Sandbox &&
		s.useBwrap == agent.useBwrap() &&
		s.devcontainerWorkspace == agent.DevcontainerWorkspace &&
		s.configDir == req.ConfigDir {
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

	bin := agent.binaryPath()
	bindDir := harnessConfigBindDir("pi", req.ConfigDir)
	var cmd *exec.Cmd
	if agent.useBwrap() {
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
		wrappedBin, wrappedArgs := WrapWithBwrap(bwrap.Path, bin, args, wd, ".pi", bindDir)
		cmd = exec.Command(wrappedBin, wrappedArgs...)
	} else if agent.useDevcontainer() {
		cmd = devcontainerExecCommand(context.Background(), agent.DevcontainerWorkspace, bin, args)
	} else {
		cmd = exec.Command(bin, args...)
		if wd != "" {
			cmd.Dir = wd
		}
	}
	applyCmdEnv(cmd, "pi", req.ConfigDir, req.Env, req.UserScopeHarness, req.NuiSessionID, req.RunID)

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
	s.binaryPath = bin
	s.sandbox = agent.Sandbox
	s.useBwrap = agent.useBwrap()
	s.devcontainerWorkspace = agent.DevcontainerWorkspace
	s.configDir = req.ConfigDir
	return nil
}

func (s *persistentPiSession) writeStdin(payload []byte) error {
	if s.stdin == nil {
		return errPiProcessEnded
	}
	_, err := s.stdin.Write(payload)
	return err
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
