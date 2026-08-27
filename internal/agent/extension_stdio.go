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
	"sync/atomic"

	"nui/internal/extensions"
	"nui/internal/store"
)

// StdioHarnessAgent implements Agent via JSON-RPC 2.0 on a harness subprocess stdin/stdout.
type StdioHarnessAgent struct {
	agentName  string
	harnessID  string
	extName    string
	mu         sync.Mutex
	rpc        *stdioHarnessRPC
	extDir     string
	runtime    extensions.RuntimeConfig
	projectID  string
}

type stdioHarnessRPC struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Scanner
	closed bool
}

var harnessRPCID atomic.Int64

func newStdioHarnessAgent(name, harnessID, projectID, extName, extDir string, rt extensions.RuntimeConfig) (*StdioHarnessAgent, error) {
	a := &StdioHarnessAgent{
		agentName: name,
		harnessID: harnessID,
		extName:   extName,
		extDir:    extDir,
		runtime:   rt,
		projectID: projectID,
	}
	if err := a.ensureRPC(RunRequest{NuiSessionID: projectID}); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *StdioHarnessAgent) Name() string { return a.agentName }

func (a *StdioHarnessAgent) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.rpc != nil {
		a.closeRPC()
	}
}

func (a *StdioHarnessAgent) buildHarnessEnv(req RunRequest) []string {
	sessionID := a.projectID
	if req.NuiSessionID != "" {
		sessionID = req.NuiSessionID
	}
	apiURL := defaultnuiAPIURL()
	if req.Env != nil {
		if v := strings.TrimSpace(req.Env[EnvnuiAPIURL]); v != "" {
			apiURL = strings.TrimRight(v, "/")
		}
	}
	fixed := map[string]string{
		"NUI_EXTENSION_DIR": a.extDir,
		"NUI_SESSION_ID":    sessionID,
		"NUI_HARNESS_ID":    a.harnessID,
		"NUI_API_URL":       apiURL,
	}
	if runID := strings.TrimSpace(req.RunID); runID != "" {
		fixed["NUI_RUN_ID"] = runID
	}
	if sdkDir, err := extensions.HitlSDKDir(); err == nil && sdkDir != "" {
		fixed["NUI_HITL_SDK_DIR"] = sdkDir
		pyPath := sdkDir
		if existing := os.Getenv("PYTHONPATH"); existing != "" {
			pyPath = sdkDir + string(os.PathListSeparator) + existing
		}
		fixed["PYTHONPATH"] = pyPath
	}
	adl := map[string]string{}
	for k, v := range req.Env {
		switch k {
		case EnvNuiSessionID, EnvnuiRunID, EnvnuiAPIURL:
			continue
		}
		adl[k] = v
	}
	// Precedence: process → secrets → per-ext → NUI_* → ADL
	return store.ExtensionProcessEnv(a.extName, fixed, adl)
}

func (a *StdioHarnessAgent) ensureRPC(req RunRequest) error {
	if a.rpc != nil && !a.rpc.closed {
		return nil
	}
	command := extensionsExpandCommand(a.runtime.Command, a.extDir)
	if len(command) == 0 {
		return fmt.Errorf("harness %s: empty runtime command", a.agentName)
	}
	cwd := a.extDir
	if c := a.runtime.Cwd; c != "" && c != "." {
		cwd = extensionsExpandCommand([]string{c}, a.extDir)[0]
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = cwd
	cmd.Env = a.buildHarnessEnv(req)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start harness %s: %w", a.agentName, err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	a.rpc = &stdioHarnessRPC{cmd: cmd, stdin: stdin, reader: scanner}
	return nil
}

func (a *StdioHarnessAgent) closeRPC() {
	if a.rpc == nil || a.rpc.closed {
		return
	}
	a.rpc.closed = true
	_ = a.rpcCall("harness.shutdown", map[string]any{}, nil)
	_ = a.rpc.stdin.Close()
	if a.rpc.cmd.Process != nil {
		_ = a.rpc.cmd.Process.Kill()
	}
	_, _ = a.rpc.cmd.Process.Wait()
	a.rpc = nil
}

func (a *StdioHarnessAgent) rpcCall(method string, params any, result any) error {
	if a.rpc == nil || a.rpc.closed {
		return fmt.Errorf("harness process not running")
	}
	id := harnessRPCID.Add(1)
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := a.rpc.stdin.Write(append(data, '\n')); err != nil {
		return err
	}
	for a.rpc.reader.Scan() {
		var msg struct {
			ID     int64          `json:"id"`
			Method string         `json:"method"`
			Params map[string]any `json:"params"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(a.rpc.reader.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Method == "harness.event" && msg.Params != nil {
			continue
		}
		if msg.ID != id {
			continue
		}
		if msg.Error != nil {
			return fmt.Errorf("%s: %s", method, msg.Error.Message)
		}
		if result != nil && len(msg.Result) > 0 {
			return json.Unmarshal(msg.Result, result)
		}
		return nil
	}
	return fmt.Errorf("%s: process ended", method)
}

func (a *StdioHarnessAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	a.mu.Lock()
	if req.NuiSessionID == "" {
		req.NuiSessionID = a.projectID
	}
	if a.rpc != nil && !a.rpc.closed {
		a.closeRPC()
	}
	if err := a.ensureRPC(req); err != nil {
		a.mu.Unlock()
		return err
	}
	rpc := a.rpc
	a.mu.Unlock()

	id := harnessRPCID.Add(1)
	runID := fmt.Sprintf("run-%d", id)
	params := map[string]any{
		"message":   req.Message,
		"runId":     runID,
		"harnessId": a.harnessID,
	}
	if req.SessionID != "" {
		params["sessionId"] = req.SessionID
	}
	if req.WorkingDir != "" {
		params["workingDir"] = req.WorkingDir
	}
	if req.SystemPrompt != "" {
		params["systemPrompt"] = req.SystemPrompt
	}
	if req.Model != "" {
		params["model"] = req.Model
	}

	reqBody, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "harness.run",
		"params":  params,
	})

	type rpcMsg struct {
		ID     *int64         `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	resultCh := make(chan rpcMsg, 1)
	done := make(chan struct{})

	a.mu.Lock()
	stdin := rpc.stdin
	reader := rpc.reader
	a.mu.Unlock()

	go func() {
		defer close(done)
		if _, err := stdin.Write(append(reqBody, '\n')); err != nil {
			return
		}
		for reader.Scan() {
			var m rpcMsg
			if err := json.Unmarshal(reader.Bytes(), &m); err != nil {
				continue
			}
			if m.Method == "harness.event" && m.Params != nil {
				if ev, ok := eventFromHarnessParams(m.Params); ok {
					events <- ev
				}
				continue
			}
			if m.ID != nil && *m.ID == id {
				resultCh <- m
				return
			}
		}
	}()

	select {
	case resp := <-resultCh:
		if resp.Error != nil {
			return fmt.Errorf("harness %s: %s", a.agentName, resp.Error.Message)
		}
		return nil
	case <-ctx.Done():
		_ = a.rpcCall("harness.cancel", map[string]any{"runId": runID}, nil)
		return ctx.Err()
	case <-done:
		select {
		case resp := <-resultCh:
			if resp.Error != nil {
				return fmt.Errorf("harness %s: %s", a.agentName, resp.Error.Message)
			}
			return nil
		default:
			return fmt.Errorf("harness %s: connection closed unexpectedly", a.agentName)
		}
	}
}

func extensionsExpandCommand(cmd []string, extDir string) []string {
	out := make([]string, len(cmd))
	for i, part := range cmd {
		out[i] = part
		if part == "${NUI_EXTENSION_DIR}" {
			out[i] = extDir
		}
	}
	return out
}
