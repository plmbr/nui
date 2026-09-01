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
	"time"

	"nui/internal/hitl"
)

var errAntigravityProcessEnded = errors.New("antigravity process ended unexpectedly")

type persistentAntigravitySession struct {
	mu sync.Mutex

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner

	stderrMu    sync.Mutex
	stderrLines []string

	binaryPath            string
	model                 string
	workingDir            string
	sandbox               string
	useBwrap              bool
	devcontainerWorkspace string
	configDir             string
	harnessPermissions    string

	conversationID string
}

func (s *persistentAntigravitySession) matches(agent *AntigravityAgent, req RunRequest) bool {
	if s == nil || s.cmd == nil || !processAlive(s.cmd) {
		return false
	}
	return s.binaryPath == agent.binaryPath() &&
		s.model == req.Model &&
		s.sandbox == agent.Sandbox &&
		s.useBwrap == agent.useBwrap() &&
		s.devcontainerWorkspace == agent.DevcontainerWorkspace &&
		s.workingDir == req.WorkingDir &&
		s.configDir == req.ConfigDir &&
		s.harnessPermissions == req.HarnessPermissions
}

func (s *persistentAntigravitySession) runTurn(ctx context.Context, agent *AntigravityAgent, req RunRequest, events chan<- Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if resume := harnessResumeSessionID(req); resume != "" && s.conversationID == "" {
		s.conversationID = resume
	}

	if err := s.ensureProcess(ctx, agent, req); err != nil {
		return err
	}

	sessionID, err := s.promptTurn(ctx, req.Message, events)
	if err != nil {
		if !errors.Is(err, errAntigravityProcessEnded) {
			return err
		}
		// Auth / config failures exit immediately; restarting just duplicates the same stderr.
		if isAntigravityFatalStderr(err.Error()) || isAntigravityFatalStderr(s.stderrSummary()) {
			if msg := s.stderrSummary(); msg != "" {
				return fmt.Errorf("%s", msg)
			}
			return err
		}
		fmt.Fprintln(os.Stderr, "[antigravity] process ended during turn, restarting")
		s.stopLocked()
		if err := s.ensureProcess(ctx, agent, req); err != nil {
			return err
		}
		sessionID, err = s.promptTurn(ctx, req.Message, events)
		if err != nil {
			if errors.Is(err, errAntigravityProcessEnded) {
				if msg := s.stderrSummary(); msg != "" {
					return fmt.Errorf("%s", msg)
				}
			}
			return err
		}
	}

	if sessionID == "" {
		sessionID = s.conversationID
	}
	events <- Event{Type: EventDone, SessionID: sessionID}
	return nil
}

func (s *persistentAntigravitySession) promptTurn(ctx context.Context, message string, events chan<- Event) (string, error) {
	if s.stdin == nil || s.stdout == nil {
		return "", errAntigravityProcessEnded
	}

	payload, err := json.Marshal(map[string]any{
		"event": "user",
		"message": map[string]any{
			"content": message,
		},
	})
	if err != nil {
		return "", err
	}
	payload = append(payload, '\n')
	if err := s.writeStdin(payload); err != nil {
		s.stopLocked()
		return "", errAntigravityProcessEnded
	}

	parser := newAntigravityStreamParser()
	if s.conversationID != "" {
		parser.conversationID = s.conversationID
	}

	for {
		if err := ctx.Err(); err != nil {
			s.stopLocked()
			return parser.conversationID, err
		}
		if !s.stdout.Scan() {
			// Give the stderr reader a moment to capture the exit reason.
			time.Sleep(50 * time.Millisecond)
			msg := s.stderrSummary()
			s.stopLocked()
			if msg != "" {
				return parser.conversationID, fmt.Errorf("%w: %s", errAntigravityProcessEnded, msg)
			}
			return parser.conversationID, errAntigravityProcessEnded
		}
		line := s.stdout.Bytes()
		if len(line) == 0 {
			continue
		}
		parser.handleLine(line, events)
		if parser.conversationID != "" {
			s.conversationID = parser.conversationID
		}
		if parser.turnDone {
			if parser.turnError != "" {
				errMsg := enrichAntigravityError(parser.turnError, s.stderrSummary())
				events <- Event{Type: EventError, Error: errMsg}
				// Persist a visible failure line in the transcript (RUN_ERROR alone is not saved).
				events <- Event{Type: EventText, Content: errMsg}
				return s.conversationID, nil
			}
			return s.conversationID, nil
		}
	}
}

func (s *persistentAntigravitySession) stderrSummary() string {
	s.stderrMu.Lock()
	defer s.stderrMu.Unlock()
	var useful []string
	for _, line := range s.stderrLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Drop noisy google.Init log lines; keep actionable messages.
		if strings.Contains(line, "logging before google.Init") {
			continue
		}
		useful = append(useful, line)
	}
	if len(useful) == 0 {
		return ""
	}
	if len(useful) > 3 {
		useful = useful[len(useful)-3:]
	}
	return strings.Join(useful, "\n")
}

func isAntigravityFatalStderr(msg string) bool {
	lower := strings.ToLower(msg)
	return strings.Contains(lower, "gemini_api_key") ||
		strings.Contains(lower, "authentication required") ||
		strings.Contains(lower, "modelprovider")
}

// enrichAntigravityError replaces opaque stream errors with a clearer stderr excerpt when available.
func enrichAntigravityError(streamErr, stderr string) string {
	streamErr = strings.TrimSpace(streamErr)
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return streamErr
	}
	generic := streamErr == "" ||
		strings.EqualFold(streamErr, "Agent execution terminated due to error.") ||
		strings.HasPrefix(strings.ToLower(streamErr), "antigravity turn failed")
	if !generic {
		return streamErr
	}
	lower := strings.ToLower(stderr)
	for _, marker := range []string{
		"quota exceeded",
		"resource_exhausted",
		"api key not valid",
		"authentication required",
		"permission denied",
		"error 429",
		"error 400",
		"error 401",
		"error 403",
	} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			// Return a window around the first useful marker.
			start := idx
			for start > 0 && stderr[start] != '\n' {
				start--
			}
			if stderr[start] == '\n' {
				start++
			}
			end := idx + len(marker)
			for end < len(stderr) && stderr[end] != '\n' {
				end++
			}
			excerpt := strings.TrimSpace(stderr[start:end])
			if len(excerpt) > 500 {
				excerpt = excerpt[:500] + "…"
			}
			if streamErr != "" {
				return streamErr + "\n" + excerpt
			}
			return excerpt
		}
	}
	return streamErr
}

func (s *persistentAntigravitySession) ensureProcess(ctx context.Context, agent *AntigravityAgent, req RunRequest) error {
	if s.matches(agent, req) {
		return nil
	}
	s.stopLocked()
	return s.start(ctx, agent, req)
}

func (s *persistentAntigravitySession) start(ctx context.Context, agent *AntigravityAgent, req RunRequest) error {
	bin := agent.binaryPath()
	useBwrap := agent.useBwrap()

	wd := req.WorkingDir
	if wd == "" {
		if cwd, err := os.Getwd(); err == nil {
			wd = cwd
		}
	}

	args := []string{
		"--input-format", "stream-json",
		"--output-format", "stream-json",
	}
	if req.HarnessPermissions != hitl.PermissionsInteractive {
		args = append(args, "--dangerously-skip-permissions")
	}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if wd != "" {
		args = append(args, "--add-dir", wd)
	}
	resume := s.conversationID
	if resume == "" {
		resume = harnessResumeSessionID(req)
	}
	if resume != "" {
		args = append(args, "--conversation", resume)
	}

	var cmd *exec.Cmd
	if useBwrap {
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
		wrappedBin, wrappedArgs := WrapWithBwrap(bwrap.Path, bin, args, wd, ".gemini", req.ConfigDir)
		cmd = exec.Command(wrappedBin, wrappedArgs...)
	} else if agent.useDevcontainer() {
		cmd = devcontainerExecCommand(ctx, agent.DevcontainerWorkspace, bin, args)
	} else {
		cmd = exec.Command(bin, args...)
		if wd != "" {
			cmd.Dir = wd
		}
	}
	applyCmdEnv(cmd, "antigravity", req.ConfigDir, antigravityCmdEnv(req.Env), req.UserScopeHarness, req.NuiSessionID, req.RunID)

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
		fmt.Fprintf(os.Stderr, "[antigravity] start error: %v\n", err)
		return err
	}

	s.stderrMu.Lock()
	s.stderrLines = nil
	s.stderrMu.Unlock()
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			text := scanner.Text()
			s.stderrMu.Lock()
			s.stderrLines = append(s.stderrLines, text)
			s.stderrMu.Unlock()
			fmt.Fprintf(os.Stderr, "[antigravity stderr] %s\n", text)
		}
	}()

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	s.cmd = cmd
	s.stdin = stdin
	s.stdout = sc
	s.binaryPath = bin
	s.model = req.Model
	s.workingDir = wd
	s.sandbox = agent.Sandbox
	s.useBwrap = useBwrap
	s.devcontainerWorkspace = agent.DevcontainerWorkspace
	s.configDir = req.ConfigDir
	s.harnessPermissions = req.HarnessPermissions
	if resume != "" {
		s.conversationID = resume
	}
	return nil
}

func (s *persistentAntigravitySession) writeStdin(payload []byte) error {
	if s.stdin == nil {
		return errAntigravityProcessEnded
	}
	_, err := s.stdin.Write(payload)
	return err
}

func (s *persistentAntigravitySession) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *persistentAntigravitySession) stopLocked() {
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

// antigravityCmdEnv ensures GEMINI_API_KEY is present for agy when modelProvider=gemini.
// agy only reads GEMINI_API_KEY (not GOOGLE_API_KEY); resolve from ADL env, process env,
// or ~/.nui/secrets.json, and alias GOOGLE_API_KEY when needed.
func antigravityCmdEnv(adlEnv map[string]string) map[string]string {
	out := make(map[string]string, len(adlEnv)+1)
	for k, v := range adlEnv {
		out[k] = v
	}
	if strings.TrimSpace(out["GEMINI_API_KEY"]) == "" {
		if key := lookupCredentialValue("GEMINI_API_KEY", adlEnv); key != "" {
			out["GEMINI_API_KEY"] = key
		} else if key := lookupCredentialValue("GOOGLE_API_KEY", adlEnv); key != "" {
			out["GEMINI_API_KEY"] = key
		}
	}
	return out
}
