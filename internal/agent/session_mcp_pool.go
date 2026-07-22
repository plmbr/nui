// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"nui/internal/mcpclient"
	"nui/internal/model"
)

type sessionMCPEntry struct {
	client     *mcpclient.Client
	configHash string
}

// GetOrConnectSessionMCP returns a session-scoped MCP client, reusing an existing
// connection when the server list is unchanged.
func (m *Manager) GetOrConnectSessionMCP(ctx context.Context, sessionID string, servers []model.ADLMCPServer) (*mcpclient.Client, []string) {
	if sessionID == "" {
		client := mcpclient.New()
		return client, client.ConnectServers(ctx, servers)
	}
	hash := serversConfigHash(servers)
	m.mcpMu.Lock()
	if entry, ok := m.mcpClients[sessionID]; ok && entry.configHash == hash && entry.client != nil {
		client := entry.client
		m.mcpMu.Unlock()
		return client, nil
	}
	if entry, ok := m.mcpClients[sessionID]; ok && entry.client != nil {
		entry.client.Close()
	}
	m.mcpMu.Unlock()

	client := mcpclient.New()
	failures := client.ConnectServers(ctx, servers)
	m.mcpMu.Lock()
	if prev, ok := m.mcpClients[sessionID]; ok && prev.client != nil && prev.client != client {
		prev.client.Close()
	}
	m.mcpClients[sessionID] = &sessionMCPEntry{client: client, configHash: hash}
	m.mcpMu.Unlock()
	return client, failures
}

// SessionMCPClient returns the active MCP client for a session, if any.
func (m *Manager) SessionMCPClient(sessionID string) *mcpclient.Client {
	m.mcpMu.Lock()
	defer m.mcpMu.Unlock()
	if entry, ok := m.mcpClients[sessionID]; ok {
		return entry.client
	}
	return nil
}

// EvictSessionMCP closes and removes the session MCP client.
func (m *Manager) EvictSessionMCP(sessionID string) {
	m.mcpMu.Lock()
	entry, ok := m.mcpClients[sessionID]
	if ok {
		delete(m.mcpClients, sessionID)
	}
	m.mcpMu.Unlock()
	if ok && entry.client != nil {
		entry.client.Close()
	}
}

// EvictAllSessionMCP closes all session MCP clients.
func (m *Manager) EvictAllSessionMCP() {
	m.mcpMu.Lock()
	entries := m.mcpClients
	m.mcpClients = map[string]*sessionMCPEntry{}
	m.mcpMu.Unlock()
	for _, entry := range entries {
		if entry.client != nil {
			entry.client.Close()
		}
	}
}

func serversConfigHash(servers []model.ADLMCPServer) string {
	b, _ := json.Marshal(servers)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
