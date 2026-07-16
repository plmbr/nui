// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"

	"loop/internal/mentions"
	"loop/internal/model"
)

// ProgrammaticRPC is a long-lived stdio JSON-RPC connection to a programmatic extension process.
type ProgrammaticRPC struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Scanner
	closed bool
}

var programmaticRPCID atomic.Int64

func StartProgrammaticRPC(command []string, env []string, dir string) (*ProgrammaticRPC, error) {
	if len(command) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = dir
	if len(env) > 0 {
		cmd.Env = env
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	return &ProgrammaticRPC{cmd: cmd, stdin: stdin, reader: scanner}, nil
}

func (c *ProgrammaticRPC) Call(method string, params any, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("rpc process closed")
	}
	id := programmaticRPCID.Add(1)
	req := rpcRequest{JSONRPC: "2.0", ID: id, Method: method, Params: params}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return err
	}
	for c.reader.Scan() {
		var resp rpcResponse
		if err := json.Unmarshal(c.reader.Bytes(), &resp); err != nil {
			continue
		}
		if resp.ID != id {
			continue
		}
		if resp.Error != nil {
			return fmt.Errorf("%s: %s", method, resp.Error.Message)
		}
		if result != nil && len(resp.Result) > 0 {
			if err := json.Unmarshal(resp.Result, result); err != nil {
				return err
			}
		}
		return nil
	}
	if err := c.reader.Err(); err != nil {
		return err
	}
	return fmt.Errorf("%s: process ended without response", method)
}

func (c *ProgrammaticRPC) Close() error {
	_ = c.Call("extension.shutdown", map[string]any{}, nil)
	return c.killProcess()
}

func (c *ProgrammaticRPC) killProcess() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.stdin != nil {
		_ = c.stdin.Close()
	}
	if c.cmd != nil && c.cmd.Process != nil {
		_ = c.cmd.Process.Kill()
		_, _ = c.cmd.Process.Wait()
	}
	return nil
}

type harnessEventHandler func(params map[string]any)

func (c *ProgrammaticRPC) RunHarness(ctx context.Context, params map[string]any, onEvent harnessEventHandler) error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("rpc process closed")
	}
	id := programmaticRPCID.Add(1)
	req := map[string]any{"jsonrpc": "2.0", "id": id, "method": "harness.run", "params": params}
	data, err := json.Marshal(req)
	if err != nil {
		c.mu.Unlock()
		return err
	}
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		c.mu.Unlock()
		return err
	}
	reader := c.reader
	c.mu.Unlock()

	type rpcMsg struct {
		ID     *int64         `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	resultCh := make(chan error, 1)
	done := make(chan struct{})

	go func() {
		defer close(done)
		for reader.Scan() {
			var m rpcMsg
			if err := json.Unmarshal(reader.Bytes(), &m); err != nil {
				continue
			}
			if m.Method == "harness.event" && m.Params != nil && onEvent != nil {
				onEvent(m.Params)
				continue
			}
			if m.ID != nil && *m.ID == id {
				if m.Error != nil {
					resultCh <- fmt.Errorf("harness.run: %s", m.Error.Message)
				} else {
					resultCh <- nil
				}
				return
			}
		}
		if err := reader.Err(); err != nil {
			resultCh <- err
			return
		}
		resultCh <- fmt.Errorf("harness.run: process ended without response")
	}()

	select {
	case err := <-resultCh:
		return err
	case <-ctx.Done():
		_ = c.Call("harness.cancel", map[string]any{"runId": params["runId"]}, nil)
		<-done
		return ctx.Err()
	}
}

func (h *programmaticHost) ListMentions(ctx context.Context, providerID string, req mentions.ListRequest) (mentions.ListResponse, error) {
	_ = ctx
	var result struct {
		Items      []mentions.Item       `json:"items"`
		Breadcrumb []mentions.Breadcrumb `json:"breadcrumb"`
	}
	params := map[string]any{
		"providerId": providerID,
		"parent":     req.Parent,
		"query":      req.Query,
		"limit":      mentions.NormalizeLimit(req.Limit),
		"workingDir": req.WorkingDir,
		"sessionId":  req.SessionID,
	}
	if err := h.rpc.Call("mention.list", params, &result); err != nil {
		return mentions.ListResponse{}, err
	}
	if result.Items == nil {
		result.Items = []mentions.Item{}
	}
	return mentions.ListResponse{Items: result.Items, Breadcrumb: result.Breadcrumb}, nil
}

func (h *programmaticHost) ResolveMention(ctx context.Context, providerID string, req mentions.ResolveRequest) (string, error) {
	_ = ctx
	var result struct {
		Text string `json:"text"`
	}
	params := map[string]any{
		"providerId": providerID,
		"value":      req.Value,
		"workingDir": req.WorkingDir,
		"sessionId":  req.SessionID,
	}
	if err := h.rpc.Call("mention.resolve", params, &result); err != nil {
		return "", err
	}
	return result.Text, nil
}

func (h *programmaticHost) DeliverHITL(channelID string, request map[string]any, workingDir, sessionID string) error {
	var result struct {
		OK bool `json:"ok"`
	}
	params := map[string]any{
		"channelId":  channelID,
		"request":    request,
		"workingDir": workingDir,
		"sessionId":  sessionID,
	}
	if err := h.rpc.Call("hitl.deliver", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("hitl.deliver: not ok")
	}
	return nil
}

func (h *programmaticHost) InvokeDeploy(req DeployRequest) (DeployResponse, error) {
	var resp DeployResponse
	if err := h.rpc.Call("extension.deploy", req, &resp); err != nil {
		return DeployResponse{}, err
	}
	if !resp.OK {
		msg := resp.Error
		if msg == "" {
			msg = resp.Message
		}
		if msg == "" {
			msg = "deploy failed"
		}
		return resp, fmt.Errorf("%s", msg)
	}
	return resp, nil
}

func (h *programmaticHost) ReadSession(ctx context.Context, handlerID, sessionID, agentType, workingDir string) ([]model.ChatMessage, string, error) {
	_ = ctx
	var result struct {
		Messages       []model.ChatMessage `json:"messages"`
		AgentSessionID string              `json:"agentSessionId"`
	}
	params := map[string]any{
		"handlerId": handlerID, "sessionId": sessionID, "agentType": agentType, "workingDir": workingDir,
	}
	if err := h.rpc.Call("storage.session.read", params, &result); err != nil {
		return nil, "", err
	}
	if result.Messages == nil {
		result.Messages = []model.ChatMessage{}
	}
	return result.Messages, result.AgentSessionID, nil
}

func (h *programmaticHost) WriteSession(ctx context.Context, handlerID, sessionID, agentType, agentSessionID, workingDir string, messages []model.ChatMessage) error {
	_ = ctx
	var result struct{ OK bool `json:"ok"` }
	params := map[string]any{
		"handlerId": handlerID, "sessionId": sessionID, "agentType": agentType,
		"agentSessionId": agentSessionID, "workingDir": workingDir, "messages": messages,
	}
	if err := h.rpc.Call("storage.session.write", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.session.write: not ok")
	}
	return nil
}

func (h *programmaticHost) DeleteSession(ctx context.Context, handlerID, sessionID, agentType, agentSessionID, workingDir string) error {
	_ = ctx
	var result struct{ OK bool `json:"ok"` }
	params := map[string]any{
		"handlerId": handlerID, "sessionId": sessionID, "agentType": agentType,
		"agentSessionId": agentSessionID, "workingDir": workingDir,
	}
	if err := h.rpc.Call("storage.session.delete", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.session.delete: not ok")
	}
	return nil
}

func (h *programmaticHost) ReadAgentMemory(ctx context.Context, handlerID, agentID string) (string, error) {
	_ = ctx
	var result struct{ Content string `json:"content"` }
	if err := h.rpc.Call("storage.agentMemory.read", map[string]any{"handlerId": handlerID, "agentId": agentID}, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

func (h *programmaticHost) WriteAgentMemory(ctx context.Context, handlerID, agentID, content, writeMode string) error {
	_ = ctx
	var result struct{ OK bool `json:"ok"` }
	params := map[string]any{"handlerId": handlerID, "agentId": agentID, "content": content, "writeMode": writeMode}
	if err := h.rpc.Call("storage.agentMemory.write", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.agentMemory.write: not ok")
	}
	return nil
}

func (h *programmaticHost) DeleteAgentMemory(ctx context.Context, handlerID, agentID string) error {
	_ = ctx
	var result struct{ OK bool `json:"ok"` }
	if err := h.rpc.Call("storage.agentMemory.delete", map[string]any{"handlerId": handlerID, "agentId": agentID}, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.agentMemory.delete: not ok")
	}
	return nil
}

func (h *programmaticHost) ReadUserMemory(ctx context.Context, handlerID string) (string, error) {
	_ = ctx
	var result struct{ Content string `json:"content"` }
	if err := h.rpc.Call("storage.userMemory.read", map[string]any{"handlerId": handlerID}, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

func (h *programmaticHost) WriteUserMemory(ctx context.Context, handlerID, content, writeMode string) error {
	_ = ctx
	var result struct{ OK bool `json:"ok"` }
	params := map[string]any{"handlerId": handlerID, "content": content, "writeMode": writeMode}
	if err := h.rpc.Call("storage.userMemory.write", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.userMemory.write: not ok")
	}
	return nil
}

func (h *programmaticHost) DeleteUserMemory(ctx context.Context, handlerID string) error {
	_ = ctx
	var result struct{ OK bool `json:"ok"` }
	if err := h.rpc.Call("storage.userMemory.delete", map[string]any{"handlerId": handlerID}, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.userMemory.delete: not ok")
	}
	return nil
}

// ProgrammaticHarnessAgent runs harnesses via a shared programmatic host connection.
type ProgrammaticHarnessAgent struct {
	host      *programmaticHost
	agentName string
	harnessID string
	projectID string
}

func NewProgrammaticHarnessAgent(host *programmaticHost, agentName, harnessID, projectID string) *ProgrammaticHarnessAgent {
	return &ProgrammaticHarnessAgent{
		host:      host,
		agentName: agentName,
		harnessID: harnessID,
		projectID: projectID,
	}
}

func (a *ProgrammaticHarnessAgent) Name() string { return a.agentName }

func (a *ProgrammaticHarnessAgent) Stop() {}

// HarnessRunEvent is one streaming event from harness.run.
type HarnessRunEvent struct {
	Type    string
	Content string
	Error   string
	Raw     map[string]any
}

func (a *ProgrammaticHarnessAgent) RunHarness(ctx context.Context, message, runID string, extra map[string]any, onEvent func(HarnessRunEvent)) error {
	params := map[string]any{
		"message":   message,
		"runId":     runID,
		"harnessId": a.harnessID,
	}
	for k, v := range extra {
		params[k] = v
	}
	return a.host.rpc.RunHarness(ctx, params, func(p map[string]any) {
		if onEvent == nil {
			return
		}
		ev := HarnessRunEvent{Raw: p}
		if t, ok := p["type"].(string); ok {
			ev.Type = t
		}
		if c, ok := p["content"].(string); ok {
			ev.Content = c
		}
		if e, ok := p["error"].(string); ok {
			ev.Error = e
		}
		onEvent(ev)
	})
}
