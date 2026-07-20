// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAnthropicCompletion(t *testing.T) {
	var captured map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		if r.Header.Get("x-api-key") != "sk-ant" {
			t.Fatalf("api key header = %q", r.Header.Get("x-api-key"))
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Fatal(err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   "msg_1",
			"type": "message",
			"role": "assistant",
			"content": []map[string]string{
				{"type": "text", "text": "hi"},
			},
		})
	}))
	defer srv.Close()

	p, err := newAnthropicProvider("sk-ant", srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	out, err := p.Completion(context.Background(), CompletionParams{
		Model:    "claude-sonnet-4-20250514",
		Messages: []Message{{Role: RoleUser, Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Choices[0].Message.Content != "hi" {
		t.Fatalf("content = %q", out.Choices[0].Message.Content)
	}
	if captured["model"] != "claude-sonnet-4-20250514" {
		t.Fatalf("request model = %v", captured["model"])
	}
}

func TestAnthropicProviderRequiresKey(t *testing.T) {
	_, err := newAnthropicProvider("", "")
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
}

func TestGeminiProviderRequiresKey(t *testing.T) {
	_, err := newGeminiProvider("", "")
	if err == nil {
		t.Fatal("expected error for missing api key")
	}
}
