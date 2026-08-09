// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type persistentOpenCodeSession struct {
	mu sync.Mutex

	sessionID string
}

func (s *persistentOpenCodeSession) runTurn(ctx context.Context, agent *OpenCodeAgent, req RunRequest, events chan<- Event) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID := s.sessionID
	if sessionID == "" {
		sessionID = req.SessionID
	}

	args := buildOpenCodeRunArgs(req, sessionID)

	bin := agent.binaryPath()
	wd := openCodeWorkingDir(req.WorkingDir)
	bindDir := harnessConfigBindDir("opencode", req.ConfigDir)
	var cmd *exec.Cmd
	if agent.useBwrap() {
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return "", fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
		wrappedBin, wrappedArgs := WrapWithBwrap(bwrap.Path, bin, args, wd, ".local/share/opencode", bindDir)
		cmd = exec.CommandContext(ctx, wrappedBin, wrappedArgs...)
	} else if agent.useDevcontainer() {
		cmd = dockerExecCommand(ctx, agent.DevcontainerContainerID, bin, args)
	} else {
		cmd = exec.CommandContext(ctx, bin, args...)
		if wd != "" {
			cmd.Dir = wd
		}
	}
	applyCmdEnv(cmd, "opencode", req.ConfigDir, req.Env, req.UserScopeHarness, req.NuiSessionID, req.RunID)
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

// buildOpenCodeRunArgs constructs CLI args for a single turn. OpenCode's --attach mode does
// not stream JSON events to stdout, so nui runs `opencode run --format json` directly and
// resumes with --session on later turns.
func buildOpenCodeRunArgs(req RunRequest, sessionID string) []string {
	args := []string{"run", "--format", "json"}
	if sessionID != "" {
		args = append(args, "--session", sessionID)
	}
	if wd := openCodeWorkingDir(req.WorkingDir); wd != "" {
		args = append(args, "--dir", wd)
	}
	if req.Model != "" {
		args = append(args, "-m", req.Model)
	}
	args = append(args, req.Message)
	return args
}

func openCodeWorkingDir(workingDir string) string {
	if workingDir != "" {
		return workingDir
	}
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	return ""
}

func (s *persistentOpenCodeSession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = ""
}
