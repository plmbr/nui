// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
)

type mcpCallToolRequest struct {
	Server    string         `json:"server"`
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

func registerMCPRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/mcp-resource", handleMCPResource)
	mux.HandleFunc("/mcp-call-tool", handleMCPCallTool)
}

func mcpConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".nui", ".mcp.json")
}

func handleMCPResource(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	serverName := r.URL.Query().Get("server")
	uri := r.URL.Query().Get("uri")
	if serverName == "" || uri == "" {
		http.Error(w, "server and uri are required", http.StatusBadRequest)
		return
	}

	html, err := mcpManager.readResource(r.Context(), serverName, uri)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cross-Origin-Embedder-Policy", "unsafe-none")
	w.Write([]byte(html))
}

func handleMCPCallTool(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req mcpCallToolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Server == "" || req.Name == "" {
		http.Error(w, "server and name are required", http.StatusBadRequest)
		return
	}
	if req.Arguments == nil {
		req.Arguments = map[string]any{}
	}

	result, err := mcpManager.callTool(r.Context(), req.Server, req.Name, req.Arguments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// bootstrapMCPLoad ensures ~/.nui/.mcp.json exists and connects configured servers.
// Called lazily on first MCP UI use, not at server startup.
func bootstrapMCPLoad(m *MCPManager) error {
	cfgPath := mcpConfigPath()
	if cfgPath == "" {
		return nil
	}
	if _, err := os.Stat(cfgPath); err != nil {
		if err := ensureMCPConfigFromClaude(cfgPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: MCP config bootstrap: %v\n", err)
		}
	}
	if err := mergeExtensionMCPConfig(cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: extension MCP merge: %v\n", err)
	}
	return m.load(cfgPath)
}

func ensureMCPConfigFromClaude(nuiCfgPath string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	claudeCfgPath := filepath.Join(home, ".claude.json")
	data, err := os.ReadFile(claudeCfgPath)
	if err != nil {
		return nil
	}
	var claudeCfg struct {
		MCPServers map[string]mcpServerConfig `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &claudeCfg); err != nil || len(claudeCfg.MCPServers) == 0 {
		return nil
	}
	out := mcpConfigFile{MCPServers: claudeCfg.MCPServers}
	encoded, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(nuiCfgPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(nuiCfgPath, encoded, 0644)
}

func mergeExtensionMCPConfig(cfgPath string) error {
	// Catalog extension MCP servers are provisioned into harness session config only.
	// They must not be merged into ~/.nui/.mcp.json — invalid stubs (e.g. python3 with
	// no args) block or pollute nui UI MCP startup.
	_ = cfgPath
	return nil
}
