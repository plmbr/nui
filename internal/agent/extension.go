// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync/atomic"
)

// ConnectionInfo mirrors the JSON written by extensions to ~/.loop/extensions/<name>.json.
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
				switch m.Params["type"] {
				case "text":
					if s, ok := m.Params["content"].(string); ok {
						events <- Event{Type: EventText, Content: s}
					}
				case "error":
					s, _ := m.Params["error"].(string)
					events <- Event{Type: EventError, Error: s}
				case "done":
					sid, _ := m.Params["sessionId"].(string)
					events <- Event{Type: EventDone, SessionID: sid}
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
