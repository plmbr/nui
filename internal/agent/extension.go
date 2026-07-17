// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// ConnectionInfo mirrors the JSON written by harness processes to ~/.nui/connections/<id>.json.
type ConnectionInfo struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	SessionID string `json:"session_id"`
	PID       int    `json:"pid"`
}

// ExtensionAgent implements Agent by speaking JSON-RPC 2.0 over TCP to a running extension process.
type ExtensionAgent struct {
	agentName string
	conn      ConnectionInfo
}

var globalRPCID atomic.Int64

func NewExtensionAgent(name string, conn ConnectionInfo) *ExtensionAgent {
	return &ExtensionAgent{agentName: name, conn: conn}
}

func (a *ExtensionAgent) Name() string { return a.agentName }

func eventFromHarnessParams(params map[string]any) (Event, bool) {
	typ, _ := params["type"].(string)
	switch typ {
	case "text":
		content, _ := params["content"].(string)
		return Event{Type: EventText, Content: content}, true
	case "error":
		errMsg, _ := params["error"].(string)
		return Event{Type: EventError, Error: errMsg}, true
	case "done":
		sid, _ := params["sessionId"].(string)
		return Event{Type: EventDone, SessionID: sid}, true
	case "tool_call_start":
		return Event{
			Type:       EventToolCallStart,
			ToolCallID: stringParam(params, "toolCallId"),
			ToolName:   stringParam(params, "toolName"),
		}, true
	case "tool_call_args":
		return Event{
			Type:       EventToolCallArgs,
			ToolCallID: stringParam(params, "toolCallId"),
			ToolArgs:   stringParam(params, "toolArgs"),
		}, true
	case "tool_call_end":
		return Event{
			Type:       EventToolCallEnd,
			ToolCallID: stringParam(params, "toolCallId"),
			ToolName:   stringParam(params, "toolName"),
			ToolArgs:   stringParam(params, "toolArgs"),
		}, true
	case "tool_call_result":
		return Event{
			Type:       EventToolCallResult,
			ToolCallID: stringParam(params, "toolCallId"),
			Content:    stringParam(params, "content"),
		}, true
	case "image":
		return Event{
			Type:           EventImage,
			ImageData:      stringParam(params, "imageData"),
			ImageMediaType: stringParam(params, "imageMediaType"),
		}, true
	case "hitl_request":
		requestID := stringParam(params, "requestId")
		if requestID == "" {
			requestID = stringParam(params, "request_id")
		}
		if requestID == "" {
			return Event{}, false
		}
		return Event{Type: EventHITLRequest, Content: requestID}, true
	default:
		return Event{}, false
	}
}

func stringParam(params map[string]any, key string) string {
	v, _ := params[key].(string)
	return v
}

// ShutdownExtension asks a running Python extension to release subprocess resources.
func ShutdownExtension(conn ConnectionInfo) {
	addr := fmt.Sprintf("%s:%d", conn.Host, conn.Port)
	tcpConn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return
	}
	defer tcpConn.Close()

	id := globalRPCID.Add(1)
	rpcReq, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "harness.shutdown",
		"params":  map[string]any{},
	})
	if _, err := tcpConn.Write(append(rpcReq, '\n')); err != nil {
		return
	}

	_ = tcpConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	scanner := bufio.NewScanner(tcpConn)
	for scanner.Scan() {
		var m struct {
			ID *int64 `json:"id"`
		}
		if json.Unmarshal(scanner.Bytes(), &m) == nil && m.ID != nil && *m.ID == id {
			return
		}
	}
}

// ShutdownHTTPAgent asks a Docker/remote HTTP agent to release subprocess resources.
func ShutdownHTTPAgent(baseURL string) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/shutdown", nil)
	if err != nil {
		return
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (a *ExtensionAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	tcpConn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", a.conn.Host, a.conn.Port))
	if err != nil {
		return fmt.Errorf("connect to extension %s: %w", a.agentName, err)
	}

	id := globalRPCID.Add(1)
	runID := fmt.Sprintf("run-%d", id)

	params := map[string]any{
		"message": req.Message,
		"runId":   runID,
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

	rpcReq, _ := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "harness.run",
		"params":  params,
	})
	tcpConn.Write(append(rpcReq, '\n'))

	type rpcMsg struct {
		ID     *int64         `json:"id"`
		Method string         `json:"method"`
		Params map[string]any `json:"params"`
		Result map[string]any `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	resultCh := make(chan rpcMsg, 1)
	done := make(chan struct{})

	scanner := bufio.NewScanner(tcpConn)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	go func() {
		defer close(done)
		for scanner.Scan() {
			var m rpcMsg
			if err := json.Unmarshal(scanner.Bytes(), &m); err != nil {
				continue
			}
			if m.Method == "harness.event" && m.Params != nil {
				if ev, ok := eventFromHarnessParams(m.Params); ok {
					events <- ev
				}
			} else if m.ID != nil && *m.ID == id {
				resultCh <- m
			}
		}
	}()

	select {
	case resp := <-resultCh:
		tcpConn.Close()
		<-done
		if resp.Error != nil {
			return fmt.Errorf("extension %s: %s", a.agentName, resp.Error.Message)
		}
		return nil
	case <-ctx.Done():
		tcpConn.Close()
		<-done
		return ctx.Err()
	case <-done:
		// Scanner ended without sending a result — drain resultCh in case of a race.
		select {
		case resp := <-resultCh:
			if resp.Error != nil {
				return fmt.Errorf("extension %s: %s", a.agentName, resp.Error.Message)
			}
			return nil
		default:
			return fmt.Errorf("extension %s: connection closed unexpectedly", a.agentName)
		}
	}
}

// httpAgentClient has no overall timeout so SSE streams can run indefinitely,
// but it times out on the response headers to catch dead servers quickly.
var httpAgentClient = &http.Client{
	Transport: &http.Transport{
		ResponseHeaderTimeout: 10 * time.Second,
	},
}

// HTTPExtensionAgent implements Agent using HTTP POST /run with SSE streaming.
// Used for Docker and remote agents.
type HTTPExtensionAgent struct {
	agentName string
	baseURL   string
}

func NewHTTPExtensionAgent(name, baseURL string) *HTTPExtensionAgent {
	return &HTTPExtensionAgent{agentName: name, baseURL: baseURL}
}

func (a *HTTPExtensionAgent) Name() string { return a.agentName }

func (a *HTTPExtensionAgent) Run(ctx context.Context, req RunRequest, events chan<- Event) error {
	params := map[string]any{"message": req.Message}
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
	if len(req.Env) > 0 {
		params["env"] = req.Env
	}
	if req.UserScopeHarness {
		params["userScopeHarness"] = true
	}
	if req.Ephemeral {
		params["ephemeral"] = true
	}

	body, _ := json.Marshal(params)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", a.baseURL+"/run", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := httpAgentClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP agent %s: %w", a.agentName, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("HTTP agent %s: status %d: %s", a.agentName, resp.StatusCode, bytes.TrimSpace(msg))
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		var params map[string]any
		if err := json.Unmarshal([]byte(line[6:]), &params); err != nil {
			continue
		}
		ev, ok := eventFromHarnessParams(params)
		if !ok {
			continue
		}
		events <- ev
		if ev.Type == EventError {
			return nil
		}
		if ev.Type == EventDone {
			return nil
		}
	}
	if err := scanner.Err(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("HTTP agent %s: %w", a.agentName, err)
	}
	return nil
}
