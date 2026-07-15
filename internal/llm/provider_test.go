// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAICompletionStream(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"finish_reason\":\"stop\"}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	p, err := newOpenAIProvider("openai", "sk-test", srv.URL+"/v1")
	if err != nil {
		t.Fatal(err)
	}
	chunks, errs := p.CompletionStream(context.Background(), CompletionParams{
		Model:    "gpt-4o-mini",
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	var text string
	for chunk := range chunks {
		if len(chunk.Choices) > 0 {
			text += chunk.Choices[0].Delta.Content
		}
	}
	if err := <-errs; err != nil {
		t.Fatal(err)
	}
	if text != "hi" {
		t.Fatalf("text = %q", text)
	}
}

func TestOllamaListModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tags" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]string{{"name": "llama3.2:latest"}},
		})
	}))
	defer srv.Close()

	p, err := newOllamaProvider(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Data) != 1 || resp.Data[0].ID != "llama3.2:latest" {
		t.Fatalf("models = %+v", resp.Data)
	}
}

func TestClassifyHTTPErrorModelNotFound(t *testing.T) {
	err := classifyHTTPError("anthropic", 404, "model not found")
	if !errors.Is(err, ErrModelNotFound) {
		t.Fatalf("err = %v", err)
	}
}

func TestNewProviderUnsupported(t *testing.T) {
	_, err := NewProvider("unknown", "key", "")
	if err == nil {
		t.Fatal("expected error")
	}
}
