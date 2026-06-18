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

// codexBinaryPaths lists locations to search for the codex binary in order.
var codexBinaryPaths = []string{
	"codex",
	"/Applications/Codex.app/Contents/Resources/codex",
}

func findCodexBinary() string {
	for _, p := range codexBinaryPaths {
		if path, err := exec.LookPath(p); err == nil {
			return path
		}
	}
	return codexBinaryPaths[0] // let exec fail with a clear error
}

// CLIAvailable reports whether the CLI required for harnessType is installed.
func CLIAvailable(harnessType string) bool {
	switch harnessType {
	case "claude-code":
		_, err := exec.LookPath("claude")
		return err == nil
	case "codex":
		for _, p := range codexBinaryPaths {
			if _, err := exec.LookPath(p); err == nil {
				return true
			}
		}
		return false
	case "pi":
		_, err := exec.LookPath("pi")
		return err == nil
	case "opencode":
		_, err := exec.LookPath("opencode")
		return err == nil
	default:
		return true
	}
}

// CodexAgent runs the Codex CLI non-interactively and streams events back.
type CodexAgent struct {
	// BinaryPath overrides the codex binary location; auto-detected if empty.
	BinaryPath string
	// Model overrides the model (e.g. "o3").
	Model string
	// Sandbox controls sandboxing: "none" disables bwrap, "bubblewrap" forces it, "" uses none.
	Sandbox string
}

func (a *CodexAgent) Name() string { return "codex" }

func (a *CodexAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	bin := a.BinaryPath
	if bin == "" {
		bin = findCodexBinary()
	}

	var args []string
	if req.SessionID != "" {
		args = []string{"exec", "resume", req.SessionID, req.Message}
	} else {
		args = []string{"exec", req.Message}
	}
	args = append(args,
		"--json",
		"--dangerously-bypass-approvals-and-sandbox",
		"--skip-git-repo-check",
		"--ignore-user-config",
	)
	if a.Model != "" {
		args = append(args, "-m", a.Model)
	}
	if req.WorkingDir != "" {
		args = append(args, "-C", req.WorkingDir)
	}

	var cmd *exec.Cmd
	if a.Sandbox == "bubblewrap" {
		bwrap := GetBwrapStatus()
		if !bwrap.Available {
			return fmt.Errorf("bubblewrap sandbox requested but not available: %s", bwrap.Error)
		}
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
		fmt.Fprintf(os.Stderr, "[codex] start error: %v\n", err)
		return err
	}

	go func() {
		s := bufio.NewScanner(stderr)
		for s.Scan() {
			fmt.Fprintf(os.Stderr, "[codex stderr] %s\n", s.Text())
		}
	}()

	var sessionID string
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var envelope struct {
			Type     string `json:"type"`
			ThreadID string `json:"thread_id"`
			Item     struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"item"`
			Error string `json:"error"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			fmt.Fprintf(os.Stderr, "[codex stdout] %s\n", line)
			continue
		}

		switch envelope.Type {
		case "thread.started":
			sessionID = envelope.ThreadID
		case "item.completed":
			if envelope.Item.Type == "agent_message" && envelope.Item.Text != "" {
				events <- Event{Type: EventText, Content: envelope.Item.Text}
			}
		case "turn.completed":
			events <- Event{Type: EventDone, SessionID: sessionID}
		case "error":
			events <- Event{Type: EventError, Error: envelope.Error}
		}
	}

	err = cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "[codex] exit code %d: %v\n", exitErr.ExitCode(), exitErr)
		} else {
			fmt.Fprintf(os.Stderr, "[codex] exit error: %v\n", err)
		}
	}
	return err
}
