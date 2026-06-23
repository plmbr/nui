// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"loop/internal/extensions"
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
	return filepath.Join(home, ".loop", ".mcp.json")
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

func initMCP() error {
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
	if _, err := os.Stat(cfgPath); err != nil {
		return nil
	}
	if err := mcpManager.load(cfgPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: MCP init: %v\n", err)
	}
	return nil
}

func ensureMCPConfigFromClaude(loopCfgPath string) error {
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
	if err := os.MkdirAll(filepath.Dir(loopCfgPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(loopCfgPath, encoded, 0644)
}

func mergeExtensionMCPConfig(cfgPath string) error {
	if extensions.Default == nil {
		return nil
	}
	extServers := extensions.Default.LoopMCPServerConfigs()
	if len(extServers) == 0 {
		return nil
	}
	var cfg mcpConfigFile
	if data, err := os.ReadFile(cfgPath); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = map[string]mcpServerConfig{}
	}
	for name, entry := range extServers {
		if _, exists := cfg.MCPServers[name]; exists {
			continue
		}
		sc := mcpServerConfig{}
		if cmd, ok := entry["command"].(string); ok {
			sc.Command = cmd
		}
		if args, ok := entry["args"].([]any); ok {
			for _, a := range args {
				if s, ok := a.(string); ok {
					sc.Args = append(sc.Args, s)
				}
			}
		}
		if sc.Command != "" {
			cfg.MCPServers[name] = sc
		}
	}
	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(cfgPath, encoded, 0644)
}
