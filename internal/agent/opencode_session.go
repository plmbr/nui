// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

type persistentOpenCodeSession struct {
	mu sync.Mutex

	server     *exec.Cmd
	baseURL    string
	workingDir string
	model      string
	sandbox    string
	useBwrap   bool
	configDir  string
	sessionID  string
}

var opencodeServeURLPattern = regexp.MustCompile(`(?i)listening on (https?://[^\s]+)`)

func (s *persistentOpenCodeSession) runTurn(ctx context.Context, agent *OpenCodeAgent, req RunRequest, events chan<- Event) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureServer(ctx, agent, req); err != nil {
		return "", err
	}

	sessionID := s.sessionID
	if sessionID == "" {
		sessionID = req.SessionID
	}

	args := []string{"run", "--format", "json", "--attach", s.baseURL}
	if sessionID != "" {
		args = append(args, "--session", sessionID)
	}
	wd := req.WorkingDir
	if wd == "" {
		if cwd, err := os.Getwd(); err == nil {
			wd = cwd
		}
	}
	if wd != "" {
		args = append(args, "--dir", wd)
	}
	if req.Model != "" {
		args = append(args, "-m", req.Model)
	}
	args = append(args, req.Message)

	bin := agent.binaryPath()
	bindDir := harnessConfigBindDir("opencode", req.ConfigDir)
	var cmd *exec.Cmd
	if agent.useBwrap() {
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return "", fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
		wrappedBin, wrappedArgs := WrapWithBwrap(bwrap.Path, bin, args, wd, ".local/share/opencode", bindDir)
		cmd = exec.CommandContext(ctx, wrappedBin, wrappedArgs...)
	} else {
		cmd = exec.CommandContext(ctx, bin, args...)
		if wd != "" {
			cmd.Dir = wd
		}
	}
	applyCmdEnv(cmd, "opencode", req.ConfigDir, req.Env, req.UserScopeHarness)
	cmd.Stdin = nil

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "[opencode stderr] %s\n", scanner.Text())
		}
	}()

	parser := newOpenCodeStreamParser()
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	latestSessionID := sessionID
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			_ = cmd.Wait()
			return "", err
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var obj struct {
			SessionID string `json:"sessionID"`
			SessionId string `json:"sessionId"`
		}
		if json.Unmarshal(line, &obj) == nil {
			if obj.SessionID != "" {
				latestSessionID = obj.SessionID
				s.sessionID = obj.SessionID
			} else if obj.SessionId != "" {
				latestSessionID = obj.SessionId
				s.sessionID = obj.SessionId
			}
		}
		parser.handleLine(line, events)
	}

	waitErr := cmd.Wait()
	if latestSessionID == "" {
		latestSessionID = s.sessionID
	}
	return latestSessionID, waitErr
}

func (s *persistentOpenCodeSession) ensureServer(ctx context.Context, agent *OpenCodeAgent, req RunRequest) error {
	wd := req.WorkingDir
	if wd == "" {
		if cwd, err := os.Getwd(); err == nil {
			wd = cwd
		}
	}
	if s.server != nil && processAlive(s.server) && s.baseURL != "" &&
		s.workingDir == wd && s.model == req.Model &&
		s.sandbox == agent.Sandbox &&
		s.useBwrap == agent.useBwrap() &&
		s.configDir == req.ConfigDir {
		return nil
	}

	s.stopLocked()
	serveArgs := []string{"serve", "--port", "0", "--hostname", "127.0.0.1"}
	bin := agent.binaryPath()
	bindDir := harnessConfigBindDir("opencode", req.ConfigDir)
	var cmd *exec.Cmd
	if agent.useBwrap() {
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
		wrappedBin, wrappedArgs := WrapWithBwrap(bwrap.Path, bin, serveArgs, wd, ".local/share/opencode", bindDir)
		cmd = exec.CommandContext(ctx, wrappedBin, wrappedArgs...)
	} else {
		cmd = exec.CommandContext(ctx, bin, serveArgs...)
		if wd != "" {
			cmd.Dir = wd
		}
	}
	applyCmdEnv(cmd, "opencode", req.ConfigDir, req.Env, req.UserScopeHarness)

	cmd.Stdin = nil

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("opencode serve start: %w", err)
	}

	s.server = cmd
	s.workingDir = wd
	s.model = req.Model
	s.sandbox = agent.Sandbox
	s.useBwrap = agent.useBwrap()
	s.configDir = req.ConfigDir

	urlCh := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintf(os.Stderr, "[opencode serve] %s\n", line)
			if match := opencodeServeURLPattern.FindStringSubmatch(line); len(match) > 1 {
				urlCh <- trimTrailingSlash(match[1])
				return
			}
		}
	}()

	select {
	case baseURL := <-urlCh:
		s.baseURL = baseURL
		return nil
	case <-time.After(15 * time.Second):
		s.stopLocked()
		return fmt.Errorf("opencode serve failed to start")
	case <-ctx.Done():
		s.stopLocked()
		return ctx.Err()
	}
}

func trimTrailingSlash(s string) string {
	for len(s) > 0 && s[len(s)-1] == '/' {
		s = s[:len(s)-1]
	}
	return s
}

func (s *persistentOpenCodeSession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *persistentOpenCodeSession) stopLocked() {
	if s.server != nil && s.server.Process != nil && s.server.ProcessState == nil {
		s.server.Process.Kill()
		_, _ = s.server.Process.Wait()
	}
	s.server = nil
	s.baseURL = ""
	s.workingDir = ""
}
