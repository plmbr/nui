// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package extensions

import (
	"context"
	"fmt"
	"os"
	"sync"

	"nui/internal/model"
)

// StorageRPCClient talks to an extension storage provider over stdio JSON-RPC.
type StorageRPCClient struct {
	rpc          *StdioRPC
	programmatic *programmaticHost
}

type storageClientCache struct {
	mu      sync.Mutex
	clients map[string]*StorageRPCClient
}

func newStorageClientCache() *storageClientCache {
	return &storageClientCache{clients: map[string]*StorageRPCClient{}}
}

func (c *storageClientCache) get(key string, factory func() (*StorageRPCClient, error)) (*StorageRPCClient, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if client, ok := c.clients[key]; ok {
		return client, nil
	}
	client, err := factory()
	if err != nil {
		return nil, err
	}
	c.clients[key] = client
	return client, nil
}

func (c *storageClientCache) closeAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, client := range c.clients {
		_ = client.Close()
	}
	c.clients = map[string]*StorageRPCClient{}
}

func NewStorageRPCClient(extDir, extName string, rt RuntimeConfig) (*StorageRPCClient, error) {
	if len(rt.Command) == 0 {
		return nil, fmt.Errorf("storage runtime command is required")
	}
	cmd := expandCommand(rt.Command, extDir)
	env := append(os.Environ(),
		"NUI_EXTENSION_DIR="+extDir,
		"NUI_EXTENSION_NAME="+extName,
	)
	rpc, err := StartStdioRPC(cmd, env, runtimeCwd(extDir, &rt))
	if err != nil {
		return nil, err
	}
	client := &StorageRPCClient{rpc: rpc}
	var info struct {
		ID string `json:"id"`
	}
	if err := client.rpc.Call("storage.info", map[string]any{}, &info); err != nil {
		_ = rpc.Close()
		return nil, err
	}
	return client, nil
}

func (c *StorageRPCClient) Close() error {
	if c.programmatic != nil {
		return nil
	}
	if c.rpc == nil {
		return nil
	}
	_ = c.rpc.Call("storage.shutdown", map[string]any{}, nil)
	return c.rpc.killProcess()
}

func (c *StorageRPCClient) ReadSession(ctx context.Context, handlerID, sessionID, agentType, workingDir string) ([]model.ChatMessage, string, error) {
	if c.programmatic != nil {
		return c.programmatic.ReadSession(ctx, handlerID, sessionID, agentType, workingDir)
	}
	_ = ctx
	var result struct {
		Messages       []model.ChatMessage `json:"messages"`
		AgentSessionID string              `json:"agentSessionId"`
	}
	params := map[string]any{
		"handlerId":  handlerID,
		"sessionId":  sessionID,
		"agentType":  agentType,
		"workingDir": workingDir,
	}
	if err := c.rpc.Call("storage.session.read", params, &result); err != nil {
		return nil, "", err
	}
	if result.Messages == nil {
		result.Messages = []model.ChatMessage{}
	}
	return result.Messages, result.AgentSessionID, nil
}

func (c *StorageRPCClient) WriteSession(ctx context.Context, handlerID, sessionID, agentType, agentSessionID, workingDir string, messages []model.ChatMessage) error {
	if c.programmatic != nil {
		return c.programmatic.WriteSession(ctx, handlerID, sessionID, agentType, agentSessionID, workingDir, messages)
	}
	_ = ctx
	var result struct {
		OK bool `json:"ok"`
	}
	params := map[string]any{
		"handlerId":      handlerID,
		"sessionId":      sessionID,
		"agentType":      agentType,
		"agentSessionId": agentSessionID,
		"workingDir":     workingDir,
		"messages":       messages,
	}
	if err := c.rpc.Call("storage.session.write", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.session.write: not ok")
	}
	return nil
}

func (c *StorageRPCClient) DeleteSession(ctx context.Context, handlerID, sessionID, agentType, agentSessionID, workingDir string) error {
	if c.programmatic != nil {
		return c.programmatic.DeleteSession(ctx, handlerID, sessionID, agentType, agentSessionID, workingDir)
	}
	_ = ctx
	var result struct {
		OK bool `json:"ok"`
	}
	params := map[string]any{
		"handlerId":      handlerID,
		"sessionId":      sessionID,
		"agentType":      agentType,
		"agentSessionId": agentSessionID,
		"workingDir":     workingDir,
	}
	if err := c.rpc.Call("storage.session.delete", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.session.delete: not ok")
	}
	return nil
}

func (c *StorageRPCClient) ReadAgentMemory(ctx context.Context, handlerID, agentID string) (string, error) {
	if c.programmatic != nil {
		return c.programmatic.ReadAgentMemory(ctx, handlerID, agentID)
	}
	_ = ctx
	var result struct {
		Content string `json:"content"`
	}
	params := map[string]any{"handlerId": handlerID, "agentId": agentID}
	if err := c.rpc.Call("storage.agentMemory.read", params, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

func (c *StorageRPCClient) WriteAgentMemory(ctx context.Context, handlerID, agentID, content, writeMode string) error {
	if c.programmatic != nil {
		return c.programmatic.WriteAgentMemory(ctx, handlerID, agentID, content, writeMode)
	}
	_ = ctx
	var result struct {
		OK bool `json:"ok"`
	}
	params := map[string]any{
		"handlerId": handlerID,
		"agentId":   agentID,
		"content":   content,
		"writeMode": writeMode,
	}
	if err := c.rpc.Call("storage.agentMemory.write", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.agentMemory.write: not ok")
	}
	return nil
}

func (c *StorageRPCClient) DeleteAgentMemory(ctx context.Context, handlerID, agentID string) error {
	if c.programmatic != nil {
		return c.programmatic.DeleteAgentMemory(ctx, handlerID, agentID)
	}
	_ = ctx
	var result struct {
		OK bool `json:"ok"`
	}
	params := map[string]any{"handlerId": handlerID, "agentId": agentID}
	if err := c.rpc.Call("storage.agentMemory.delete", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.agentMemory.delete: not ok")
	}
	return nil
}

func (c *StorageRPCClient) ReadUserMemory(ctx context.Context, handlerID string) (string, error) {
	if c.programmatic != nil {
		return c.programmatic.ReadUserMemory(ctx, handlerID)
	}
	_ = ctx
	var result struct {
		Content string `json:"content"`
	}
	params := map[string]any{"handlerId": handlerID}
	if err := c.rpc.Call("storage.userMemory.read", params, &result); err != nil {
		return "", err
	}
	return result.Content, nil
}

func (c *StorageRPCClient) WriteUserMemory(ctx context.Context, handlerID, content, writeMode string) error {
	if c.programmatic != nil {
		return c.programmatic.WriteUserMemory(ctx, handlerID, content, writeMode)
	}
	_ = ctx
	var result struct {
		OK bool `json:"ok"`
	}
	params := map[string]any{
		"handlerId": handlerID,
		"content":   content,
		"writeMode": writeMode,
	}
	if err := c.rpc.Call("storage.userMemory.write", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.userMemory.write: not ok")
	}
	return nil
}

func (c *StorageRPCClient) DeleteUserMemory(ctx context.Context, handlerID string) error {
	if c.programmatic != nil {
		return c.programmatic.DeleteUserMemory(ctx, handlerID)
	}
	_ = ctx
	var result struct {
		OK bool `json:"ok"`
	}
	params := map[string]any{"handlerId": handlerID}
	if err := c.rpc.Call("storage.userMemory.delete", params, &result); err != nil {
		return err
	}
	if !result.OK {
		return fmt.Errorf("storage.userMemory.delete: not ok")
	}
	return nil
}

// StorageClient resolves a storage RPC client for an extension.
func (r *Registry) StorageClient(extName string) (*StorageRPCClient, error) {
	r.mu.RLock()
	ext, ok := r.extensions[extName]
	r.mu.RUnlock()
	if !ok || r.isDisabled(extName) {
		return nil, fmt.Errorf("extension %q not found", extName)
	}
	if ext.programmaticHost != nil {
		return &StorageRPCClient{programmatic: ext.programmaticHost}, nil
	}
	if ext.storageRuntime == nil {
		return nil, fmt.Errorf("extension %q has no storage runtime", extName)
	}
	if r.storageCache == nil {
		r.storageCache = newStorageClientCache()
	}
	extDir := ext.Dir
	runtime := *ext.storageRuntime
	return r.storageCache.get(extName, func() (*StorageRPCClient, error) {
		return NewStorageRPCClient(extDir, extName, runtime)
	})
}
