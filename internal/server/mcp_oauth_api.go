// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"loop/internal/mcpoauth"
	"loop/internal/model"
	"loop/internal/store"
)

func registerMCPOAuthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/mcp-oauth/start", handleMCPOAuthStart)
	mux.HandleFunc("/api/mcp-oauth/callback", handleMCPOAuthCallback)
	mux.HandleFunc("/api/mcp-oauth/flow", handleMCPOAuthFlow)
	mux.HandleFunc("/api/mcp-oauth/complete", handleMCPOAuthComplete)
	mux.HandleFunc("/api/mcp-oauth/status", handleMCPOAuthStatus)
	mux.HandleFunc("/api/mcp-oauth/redirect-uri", handleMCPOAuthRedirectURI)
	mux.HandleFunc("/api/mcp-oauth/disconnect", handleMCPOAuthDisconnect)
}

func handleMCPOAuthStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		ServerKey string              `json:"serverKey"`
		Server    *model.ADLMCPServer `json:"server,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	srv, err := resolveMCPOAuthServer(body.ServerKey, body.Server)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	flowID, authURL, redirectURI, err := mcpoauth.StartFlow(r.Context(), srv)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"flowId":      flowID,
		"authUrl":     authURL,
		"redirectUri": redirectURI,
		"serverKey":   mcpoauth.ServerKey(srv),
	})
}

func handleMCPOAuthFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flowID := strings.TrimSpace(r.URL.Query().Get("flowId"))
	if flowID == "" {
		http.Error(w, "flowId is required", http.StatusBadRequest)
		return
	}
	outcome, ok := mcpoauth.FlowOutcomeByID(flowID)
	if !ok {
		http.Error(w, "unknown or expired oauth flow", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"flowId":    outcome.FlowID,
		"serverKey": outcome.ServerKey,
		"status":    string(outcome.Status),
		"error":     outcome.Error,
	})
}

func handleMCPOAuthCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	if err := mcpoauth.DeliverCallback(code, state); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!DOCTYPE html><html><head><title>Loop MCP</title></head><body><p>Authentication successful. You can close this window and return to Loop.</p><script>
(function(){
  var msg = {type:"loop-mcp-oauth-success"};
  if(window.opener){window.opener.postMessage(msg, "*");}
  try{new BroadcastChannel("loop-mcp-oauth").postMessage(msg);}catch(e){}
})();
</script></body></html>`)
}

func handleMCPOAuthComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		FlowID      string `json:"flowId"`
		CallbackURL string `json:"callbackUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := mcpoauth.DeliverCallbackURL(body.FlowID, body.CallbackURL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func handleMCPOAuthStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	serverKey := strings.TrimSpace(r.URL.Query().Get("serverKey"))
	if serverKey != "" {
		if mcpoauth.HasValidToken(serverKey) {
			writeJSON(w, http.StatusOK, map[string]any{
				"serverKey": serverKey,
				"status":    string(mcpoauth.AuthStatusConnected),
			})
			return
		}
		srv, err := lookupMCPServerByKey(serverKey)
		if err != nil {
			if status, ok := mcpoauth.TokenAuthStatus(serverKey); ok {
				writeJSON(w, http.StatusOK, map[string]any{
					"serverKey": serverKey,
					"status":    string(status),
				})
				return
			}
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"serverKey": serverKey,
			"status":    mcpoauth.StatusForServer(srv),
		})
		return
	}
	servers, err := store.LoadMCPServers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	type entry struct {
		ServerKey string `json:"serverKey"`
		Name      string `json:"name"`
		Status    string `json:"status"`
	}
	out := make([]entry, 0, len(servers))
	for _, srv := range servers {
		out = append(out, entry{
			ServerKey: mcpoauth.ServerKey(srv),
			Name:      srv.Name,
			Status:    string(mcpoauth.StatusForServer(srv)),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"servers": out})
}

func handleMCPOAuthRedirectURI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uri, err := mcpoauth.RedirectURI()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"redirectUri": uri})
}

func handleMCPOAuthDisconnect(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	serverKey := strings.TrimSpace(r.URL.Query().Get("serverKey"))
	if serverKey == "" {
		http.Error(w, "serverKey is required", http.StatusBadRequest)
		return
	}
	if err := mcpoauth.DeleteToken(serverKey); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func resolveMCPOAuthServer(serverKey string, inline *model.ADLMCPServer) (model.ADLMCPServer, error) {
	if inline != nil && strings.TrimSpace(inline.URL) != "" {
		return *inline, nil
	}
	key := strings.TrimSpace(serverKey)
	if key == "" && inline != nil {
		key = mcpoauth.ServerKey(*inline)
	}
	if key == "" {
		return model.ADLMCPServer{}, fmt.Errorf("serverKey or server.url is required")
	}
	return lookupMCPServerByKey(key)
}

func lookupMCPServerByKey(serverKey string) (model.ADLMCPServer, error) {
	servers, err := store.LoadMCPServers()
	if err != nil {
		return model.ADLMCPServer{}, err
	}
	for _, srv := range servers {
		if mcpoauth.ServerKey(srv) == serverKey || strings.TrimSpace(srv.Name) == serverKey {
			return srv, nil
		}
	}
	return model.ADLMCPServer{}, fmt.Errorf("mcp server %q not found in user catalog", serverKey)
}
