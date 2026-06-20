// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type persistentCodexSession struct {
	mu sync.Mutex

	cmd    *exec.Cmd
	stdout *bufio.Scanner

	binaryPath string
	model      string
	sandbox    string
	useBwrap   bool
	workingDir string

	threadID string
}

func (s *persistentCodexSession) matches(agent *CodexAgent, req RunRequest) bool {
	if s == nil {
		return false
	}
	return s.binaryPath == agent.binaryPath() &&
		s.model == req.Model &&
		s.sandbox == agent.Sandbox &&
		s.useBwrap == agent.useBwrap() &&
		s.workingDir == req.WorkingDir
}

func (s *persistentCodexSession) runTurn(ctx context.Context, agent *CodexAgent, req RunRequest, events chan<- Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.ensureIdle(); err != nil {
		return err
	}

	if s.workingDir != req.WorkingDir || s.model != req.Model {
		s.threadID = ""
	}

	bin := agent.binaryPath()
	args := s.buildArgs(req)
	var cmd *exec.Cmd
	if agent.useBwrap() {
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
		wrappedBin, wrappedArgs := WrapWithBwrap(bwrap.Path, bin, args, req.WorkingDir, ".codex")
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
		fmt.Fprintf(os.Stderr, "[codex] start error: %v\n", err)
		return err
	}

	s.cmd = cmd
	s.stdout = bufio.NewScanner(stdout)
	s.stdout.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	s.binaryPath = bin
	s.model = req.Model
	s.sandbox = agent.Sandbox
	s.useBwrap = agent.useBwrap()
	s.workingDir = req.WorkingDir

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			fmt.Fprintf(os.Stderr, "[codex stderr] %s\n", scanner.Text())
		}
	}()

	parser := newCodexStreamParser()
	var sessionID string
	for s.stdout.Scan() {
		if err := ctx.Err(); err != nil {
			s.stopLocked()
			return err
		}
		sid, done := parser.handleLine(s.stdout.Bytes(), events)
		if sid != "" {
			sessionID = sid
			s.threadID = sid
		}
		if done {
			events <- Event{Type: EventDone, SessionID: sessionID}
			break
		}
	}

	waitErr := cmd.Wait()
	s.cmd = nil
	s.stdout = nil
	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "[codex] exit code %d: %v\n", exitErr.ExitCode(), exitErr)
		} else {
			fmt.Fprintf(os.Stderr, "[codex] exit error: %v\n", waitErr)
		}
	}
	return waitErr
}

func (s *persistentCodexSession) buildArgs(req RunRequest) []string {
	threadID := s.threadID
	if threadID == "" {
		threadID = req.SessionID
	}

	var args []string
	if threadID != "" {
		args = []string{"exec", "resume", threadID, req.Message}
	} else {
		args = []string{"exec", req.Message}
	}
	args = append(args,
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
		"--ignore-user-config",
	)
	if baseURL := os.Getenv("OPENAI_BASE_URL"); baseURL != "" {
		args = append(args,
			"-c", `model_provider="loop_gateway"`,
			"-c", fmt.Sprintf(`model_providers.loop_gateway={name="loop_gateway",env_key="OPENAI_API_KEY",base_url="%s",supports_websockets=false}`, baseURL),
		)
	}
	if req.Model != "" {
		args = append(args, "-m", req.Model)
	}
	if req.WorkingDir != "" && threadID == "" {
		args = append(args, "-C", req.WorkingDir)
	}
	return args
}

func (s *persistentCodexSession) ensureIdle() error {
	if s.cmd == nil || !processAlive(s.cmd) {
		return nil
	}
	return fmt.Errorf("codex session is busy")
}

func (s *persistentCodexSession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *persistentCodexSession) stopLocked() {
	if s.cmd == nil || s.cmd.Process == nil {
		s.cmd = nil
		s.stdout = nil
		return
	}
	if processAlive(s.cmd) {
		s.cmd.Process.Signal(os.Interrupt)
		_ = s.cmd.Wait()
	}
	s.cmd = nil
	s.stdout = nil
}
