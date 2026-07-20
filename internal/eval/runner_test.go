// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package eval

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"nui/internal/nuiclient"
)

func TestRunnerRun_agentNotFound(t *testing.T) {
	r := &Runner{Client: nuiclient.New("http://127.0.0.1:1")}
	_, err := r.Run(context.Background(), Options{AgentID: "missing-agent"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunnerRun_noEvals(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	yaml := `adl: "1.0"
id: plain-agent
name: Plain Agent
harness:
  type: claude-code
`
	if err := os.WriteFile(filepath.Join(agentsDir, "plain-agent.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	r := &Runner{Client: nuiclient.New("http://127.0.0.1:1")}
	_, err := r.Run(context.Background(), Options{AgentID: "plain-agent"})
	if err == nil {
		t.Fatal("expected error for agent without evals")
	}
}

func TestRunnerRun_passesEvalCase(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	agentsDir := filepath.Join(home, ".nui", "agents")
	if err := os.MkdirAll(agentsDir, 0o700); err != nil {
		t.Fatal(err)
	}
	yaml := `adl: "1.0"
id: eval-agent
name: Eval Agent
harness:
  type: claude-code
evals:
  - name: smoke
    input: say hello
    expect:
      type: contains
      value: hello
`
	if err := os.WriteFile(filepath.Join(agentsDir, "eval-agent.yaml"), []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	var getRunCalls atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"id":        "sess-1",
			"name":      "eval",
			"agentType": "eval-agent",
			"workingDir": t.TempDir(),
		})
	})
	mux.HandleFunc("/api/sessions/sess-1/runs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"runId":     "run-1",
			"sessionId": "sess-1",
			"status":    "running",
		})
	})
	mux.HandleFunc("/api/sessions/sess-1/runs/run-1", func(w http.ResponseWriter, r *http.Request) {
		getRunCalls.Add(1)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"runId":     "run-1",
			"sessionId": "sess-1",
			"status":    "completed",
			"output":    "hello there",
		})
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	r := &Runner{Client: nuiclient.New(srv.URL)}
	summary, err := r.Run(context.Background(), Options{
		AgentID:    "eval-agent",
		WorkingDir: t.TempDir(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if summary.Passed != 1 || summary.Failed != 0 || summary.Errors != 0 {
		t.Fatalf("summary = %+v", summary)
	}
	if len(summary.Results) != 1 || summary.Results[0].Status != "pass" {
		t.Fatalf("results = %+v", summary.Results)
	}
}
