// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"loop/internal/agents"
	"loop/internal/eval"
	"loop/internal/loopclient"
)

func handleAgentEvalRun(w http.ResponseWriter, r *http.Request, agentID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	def, ok := agents.LookupDefinition(agentID)
	if !ok {
		http.Error(w, "agent not found", http.StatusNotFound)
		return
	}
	if len(def.Evals) == 0 {
		http.Error(w, "agent has no evals defined", http.StatusBadRequest)
		return
	}

	var body struct {
		WorkingDir string   `json:"workingDir,omitempty"`
		Cases      []string `json:"cases,omitempty"`
		Parallel   int      `json:"parallel,omitempty"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
	}

	ctx := r.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	// Eval runs can take several minutes (devcontainer, LLM judge, etc.).
	ctx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	client := loopclient.New(serverBaseURL(r))
	runner := &eval.Runner{Client: client}
	summary, err := runner.Run(ctx, eval.Options{
		AgentID:     agentID,
		WorkingDir:  body.WorkingDir,
		FilterNames: body.Cases,
		Parallel:    body.Parallel,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func serverBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
		scheme = proto
	}
	return scheme + "://" + r.Host
}
