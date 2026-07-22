// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"testing"

	"nui/internal/model"
)

func TestGetOrConnectSessionMCP_reusesClient(t *testing.T) {
	mgr := NewManager()
	defer mgr.EvictAllSessionMCP()

	servers := []model.ADLMCPServer{{Name: "empty", Command: "/nonexistent", Args: []string{}}}
	client1, _ := mgr.GetOrConnectSessionMCP(context.Background(), "sess-1", servers)
	client2, _ := mgr.GetOrConnectSessionMCP(context.Background(), "sess-1", servers)
	if client1 != client2 {
		t.Fatal("expected same client instance for unchanged server list")
	}
}

func TestGetOrConnectSessionMCP_recreatesOnConfigChange(t *testing.T) {
	mgr := NewManager()
	defer mgr.EvictAllSessionMCP()

	serversA := []model.ADLMCPServer{{Name: "a", Command: "/nonexistent-a"}}
	serversB := []model.ADLMCPServer{{Name: "b", Command: "/nonexistent-b"}}
	client1, _ := mgr.GetOrConnectSessionMCP(context.Background(), "sess-2", serversA)
	client2, _ := mgr.GetOrConnectSessionMCP(context.Background(), "sess-2", serversB)
	if client1 == client2 {
		t.Fatal("expected new client when server list changes")
	}
}

func TestEvictSessionMCP(t *testing.T) {
	mgr := NewManager()
	servers := []model.ADLMCPServer{{Name: "x", Command: "/nonexistent"}}
	mgr.GetOrConnectSessionMCP(context.Background(), "sess-3", servers)
	if mgr.SessionMCPClient("sess-3") == nil {
		t.Fatal("expected client before eviction")
	}
	mgr.EvictSessionMCP("sess-3")
	if mgr.SessionMCPClient("sess-3") != nil {
		t.Fatal("expected nil after eviction")
	}
}

func TestStopEvictsSessionMCP(t *testing.T) {
	mgr := NewManager()
	servers := []model.ADLMCPServer{{Name: "x", Command: "/nonexistent"}}
	mgr.GetOrConnectSessionMCP(context.Background(), "sess-4", servers)
	mgr.Stop("sess-4")
	if mgr.SessionMCPClient("sess-4") != nil {
		t.Fatal("expected nil after Stop")
	}
}
