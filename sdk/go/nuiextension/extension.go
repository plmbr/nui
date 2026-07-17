// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package nuiextension

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

// Extension is the override-based programmatic extension interface.
type Extension interface {
	Initialize() error
	Shutdown()
	GetHarnesses() []map[string]any
	GetAgents() []map[string]any
	GetMentionProviders() []map[string]any
	GetRules() []map[string]any
	GetHITLChannels() []map[string]any
	GetStorageHandlers() []map[string]any
	GetDeployers() []map[string]any
	RunHarness(ctx context.Context, harnessID, message string, params map[string]any, emit func(map[string]any)) error
	ListMentions(providerID, parent, query string, limit int, params map[string]any) (map[string]any, error)
	ResolveMention(providerID, value string, params map[string]any) (map[string]any, error)
	DeliverHITL(channelID string, request map[string]any, params map[string]any) (map[string]any, error)
	ReadSession(handlerID, sessionID, agentType, workingDir string, params map[string]any) (map[string]any, error)
	WriteSession(handlerID, sessionID, agentType, agentSessionID, workingDir string, messages []any, params map[string]any) (map[string]any, error)
	DeleteSession(handlerID, sessionID, agentType, agentSessionID, workingDir string, params map[string]any) (map[string]any, error)
	ReadAgentMemory(handlerID, agentID string, params map[string]any) (map[string]any, error)
	WriteAgentMemory(handlerID, agentID, content, writeMode string, params map[string]any) (map[string]any, error)
	DeleteAgentMemory(handlerID, agentID string, params map[string]any) (map[string]any, error)
	ReadUserMemory(handlerID string, params map[string]any) (map[string]any, error)
	WriteUserMemory(handlerID, content, writeMode string, params map[string]any) (map[string]any, error)
	DeleteUserMemory(handlerID string, params map[string]any) (map[string]any, error)
	Deploy(deployerID string, params map[string]any) (map[string]any, error)
}

// Base provides default empty implementations.
type Base struct {
	ExtensionDir  string
	ExtensionName string
	APIURL        string
}

func (b *Base) Initialize() error                       { return nil }
func (b *Base) Shutdown()                               {}
func (b *Base) GetHarnesses() []map[string]any          { return nil }
func (b *Base) GetAgents() []map[string]any             { return nil }
func (b *Base) GetMentionProviders() []map[string]any   { return nil }
func (b *Base) GetRules() []map[string]any              { return nil }
func (b *Base) GetHITLChannels() []map[string]any       { return nil }
func (b *Base) GetStorageHandlers() []map[string]any    { return nil }
func (b *Base) GetDeployers() []map[string]any          { return nil }
func (b *Base) RunHarness(ctx context.Context, harnessID, message string, params map[string]any, emit func(map[string]any)) error {
	return nil
}
func (b *Base) ListMentions(providerID, parent, query string, limit int, params map[string]any) (map[string]any, error) {
	return map[string]any{"items": []any{}, "breadcrumb": []any{}}, nil
}
func (b *Base) ResolveMention(providerID, value string, params map[string]any) (map[string]any, error) {
	return map[string]any{"text": value}, nil
}
func (b *Base) DeliverHITL(channelID string, request map[string]any, params map[string]any) (map[string]any, error) {
	return map[string]any{"ok": true}, nil
}
func (b *Base) ReadSession(handlerID, sessionID, agentType, workingDir string, params map[string]any) (map[string]any, error) {
	_ = handlerID, sessionID, agentType, workingDir, params
	return map[string]any{"messages": []any{}, "agentSessionId": ""}, nil
}
func (b *Base) WriteSession(handlerID, sessionID, agentType, agentSessionID, workingDir string, messages []any, params map[string]any) (map[string]any, error) {
	_ = handlerID, sessionID, agentType, agentSessionID, workingDir, messages, params
	return map[string]any{"ok": true}, nil
}
func (b *Base) DeleteSession(handlerID, sessionID, agentType, agentSessionID, workingDir string, params map[string]any) (map[string]any, error) {
	_ = handlerID, sessionID, agentType, agentSessionID, workingDir, params
	return map[string]any{"ok": true}, nil
}
func (b *Base) ReadAgentMemory(handlerID, agentID string, params map[string]any) (map[string]any, error) {
	_ = handlerID, agentID, params
	return map[string]any{"content": ""}, nil
}
func (b *Base) WriteAgentMemory(handlerID, agentID, content, writeMode string, params map[string]any) (map[string]any, error) {
	_ = handlerID, agentID, content, writeMode, params
	return map[string]any{"ok": true}, nil
}
func (b *Base) DeleteAgentMemory(handlerID, agentID string, params map[string]any) (map[string]any, error) {
	_ = handlerID, agentID, params
	return map[string]any{"ok": true}, nil
}
func (b *Base) ReadUserMemory(handlerID string, params map[string]any) (map[string]any, error) {
	_ = handlerID, params
	return map[string]any{"content": ""}, nil
}
func (b *Base) WriteUserMemory(handlerID, content, writeMode string, params map[string]any) (map[string]any, error) {
	_ = handlerID, content, writeMode, params
	return map[string]any{"ok": true}, nil
}
func (b *Base) DeleteUserMemory(handlerID string, params map[string]any) (map[string]any, error) {
	_ = handlerID, params
	return map[string]any{"ok": true}, nil
}
func (b *Base) Deploy(deployerID string, params map[string]any) (map[string]any, error) {
	return map[string]any{"ok": false, "error": "deploy not implemented"}, nil
}

// ServeStdio runs the JSON-RPC loop for ext.
func ServeStdio(ext Extension) error {
	extImpl := ext
	base := Base{
		ExtensionDir:  os.Getenv("NUI_EXTENSION_DIR"),
		ExtensionName: os.Getenv("NUI_EXTENSION_NAME"),
		APIURL:        os.Getenv("NUI_API_URL"),
	}
	_ = base
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	var mu sync.Mutex
	write := func(msg map[string]any) {
		mu.Lock()
		defer mu.Unlock()
		data, _ := json.Marshal(msg)
		fmt.Fprintf(os.Stdout, "%s\n", data)
	}
	for scanner.Scan() {
		var req map[string]any
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			continue
		}
		method, _ := req["method"].(string)
		rid := req["id"]
		params, _ := req["params"].(map[string]any)
		if params == nil {
			params = map[string]any{}
		}
		switch method {
		case "extension.initialize":
			_ = extImpl.Initialize()
			write(map[string]any{
				"jsonrpc": "2.0", "id": rid, "result": map[string]any{
					"apiVersion": "nui.plmbr.dev/extension/v1",
					"name":       os.Getenv("NUI_EXTENSION_NAME"),
					"harnesses":  extImpl.GetHarnesses(),
					"agents":     extImpl.GetAgents(),
					"mentionProviders": extImpl.GetMentionProviders(),
					"rules":      extImpl.GetRules(),
					"hitlChannels": extImpl.GetHITLChannels(),
					"storageHandlers": extImpl.GetStorageHandlers(),
					"agentDeployers": extImpl.GetDeployers(),
				},
			})
		case "extension.shutdown":
			extImpl.Shutdown()
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": map[string]any{"ok": true}})
			os.Exit(0)
		case "harness.run":
			runID, _ := params["runId"].(string)
			harnessID, _ := params["harnessId"].(string)
			message, _ := params["message"].(string)
			emit := func(ev map[string]any) {
				p := map[string]any{"runId": runID}
				for k, v := range ev {
					p[k] = v
				}
				write(map[string]any{"jsonrpc": "2.0", "method": "harness.event", "params": p})
			}
			_ = extImpl.RunHarness(context.Background(), harnessID, message, params, emit)
			emit(map[string]any{"type": "done"})
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": map[string]any{"runId": runID}})
		case "mention.list":
			providerID, _ := params["providerId"].(string)
			parent, _ := params["parent"].(string)
			query, _ := params["query"].(string)
			limit := 20
			if v, ok := params["limit"].(float64); ok {
				limit = int(v)
			}
			result, _ := extImpl.ListMentions(providerID, parent, query, limit, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "mention.resolve":
			providerID, _ := params["providerId"].(string)
			value, _ := params["value"].(string)
			result, _ := extImpl.ResolveMention(providerID, value, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "hitl.deliver":
			channelID, _ := params["channelId"].(string)
			request, _ := params["request"].(map[string]any)
			result, _ := extImpl.DeliverHITL(channelID, request, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.session.read":
			handlerID, _ := params["handlerId"].(string)
			sessionID, _ := params["sessionId"].(string)
			agentType, _ := params["agentType"].(string)
			workingDir, _ := params["workingDir"].(string)
			result, _ := extImpl.ReadSession(handlerID, sessionID, agentType, workingDir, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.session.write":
			handlerID, _ := params["handlerId"].(string)
			sessionID, _ := params["sessionId"].(string)
			agentType, _ := params["agentType"].(string)
			agentSessionID, _ := params["agentSessionId"].(string)
			workingDir, _ := params["workingDir"].(string)
			var messages []any
			if raw, ok := params["messages"].([]any); ok {
				messages = raw
			}
			result, _ := extImpl.WriteSession(handlerID, sessionID, agentType, agentSessionID, workingDir, messages, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.session.delete":
			handlerID, _ := params["handlerId"].(string)
			sessionID, _ := params["sessionId"].(string)
			agentType, _ := params["agentType"].(string)
			agentSessionID, _ := params["agentSessionId"].(string)
			workingDir, _ := params["workingDir"].(string)
			result, _ := extImpl.DeleteSession(handlerID, sessionID, agentType, agentSessionID, workingDir, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.agentMemory.read":
			handlerID, _ := params["handlerId"].(string)
			agentID, _ := params["agentId"].(string)
			result, _ := extImpl.ReadAgentMemory(handlerID, agentID, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.agentMemory.write":
			handlerID, _ := params["handlerId"].(string)
			agentID, _ := params["agentId"].(string)
			content, _ := params["content"].(string)
			writeMode, _ := params["writeMode"].(string)
			result, _ := extImpl.WriteAgentMemory(handlerID, agentID, content, writeMode, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.agentMemory.delete":
			handlerID, _ := params["handlerId"].(string)
			agentID, _ := params["agentId"].(string)
			result, _ := extImpl.DeleteAgentMemory(handlerID, agentID, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.userMemory.read":
			handlerID, _ := params["handlerId"].(string)
			result, _ := extImpl.ReadUserMemory(handlerID, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.userMemory.write":
			handlerID, _ := params["handlerId"].(string)
			content, _ := params["content"].(string)
			writeMode, _ := params["writeMode"].(string)
			result, _ := extImpl.WriteUserMemory(handlerID, content, writeMode, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "storage.userMemory.delete":
			handlerID, _ := params["handlerId"].(string)
			result, _ := extImpl.DeleteUserMemory(handlerID, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		case "extension.deploy":
			deployerID, _ := params["deployerId"].(string)
			result, _ := extImpl.Deploy(deployerID, params)
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "result": result})
		default:
			write(map[string]any{"jsonrpc": "2.0", "id": rid, "error": map[string]any{"code": -32601, "message": fmt.Sprintf("method not found: %s", method)}})
		}
	}
	if err := scanner.Err(); err != nil && err != io.EOF {
		return err
	}
	return nil
}
