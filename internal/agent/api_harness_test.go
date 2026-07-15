// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"loop/internal/llm"
	"loop/internal/model"
)

func TestAPIHarnessAvailable(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "")
	h := model.ADLHarness{Type: "api", Provider: "anthropic"}
	if APIHarnessAvailable(h) {
		t.Fatal("expected unavailable without key")
	}
	t.Setenv("ANTHROPIC_API_KEY", "sk-test")
	if !APIHarnessAvailable(h) {
		t.Fatal("expected available with key")
	}
}

func TestAPIHarnessAvailableOllama(t *testing.T) {
	h := model.ADLHarness{Type: "api", Provider: "ollama"}
	if !APIHarnessAvailable(h) {
		t.Fatal("ollama should always be listed available")
	}
}

func TestResolveAPIProviderProfileOpenRouter(t *testing.T) {
	h := model.ADLHarness{Type: "api", Provider: "openrouter"}
	p := ResolveAPIProviderProfile(h)
	if p.ProviderID != "openai" {
		t.Fatalf("provider id = %q, want openai", p.ProviderID)
	}
	if p.BaseURL != openRouterDefaultBaseURL {
		t.Fatalf("base url = %q", p.BaseURL)
	}
	if len(p.APIKeyEnvs) != 1 || p.APIKeyEnvs[0] != "OPENROUTER_API_KEY" {
		t.Fatalf("api key envs = %v", p.APIKeyEnvs)
	}
}

func TestBuildAPIMessages(t *testing.T) {
	msgs := buildAPIMessages(RunRequest{
		SystemPrompt: "You are helpful.",
		History: []model.ChatMessage{
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "hello"},
		},
		Message: "next",
	})
	if len(msgs) != 4 {
		t.Fatalf("len = %d", len(msgs))
	}
	if msgs[0].Role != "system" || msgs[3].Content != "next" {
		t.Fatalf("messages = %+v", msgs)
	}
}

func TestBuildAPIMessages_vizOnlyAssistantHistory(t *testing.T) {
	msgs := buildAPIMessages(RunRequest{
		History: []model.ChatMessage{
			{Role: "user", Content: "show a chart"},
			{
				Role: "assistant",
				Parts: []model.ChatMessagePart{
					{
						Type:                "tool",
						ToolName:            "show_visualization",
						VisualizationTitle:  "Sales",
						VisualizationHTML:   "<canvas></canvas>",
					},
				},
			},
		},
		Message: "show another chart",
	})
	if len(msgs) != 3 {
		t.Fatalf("len = %d", len(msgs))
	}
	if msgs[1].Content != "[Rendered visualization: Sales]" {
		t.Fatalf("assistant history = %q", msgs[1].Content)
	}
}

func TestAPIHarnessAvailableDef(t *testing.T) {
	def := model.ADLDefinition{
		ID:      "anthropic",
		Harness: model.ADLHarness{Type: "api", Provider: "anthropic"},
	}
	t.Setenv("ANTHROPIC_API_KEY", "")
	if APIHarnessAvailableDef(def) {
		t.Fatal("expected false")
	}
}

func TestResolveAPIKeyFromEnvMap(t *testing.T) {
	key, err := resolveAPIKey(APIProviderProfile{
		APIKeyEnvs: []string{"TEST_LOOP_API_KEY"},
		NeedsKey:   true,
	}, map[string]string{"TEST_LOOP_API_KEY": "from-adl"})
	if err != nil {
		t.Fatal(err)
	}
	if key != "from-adl" {
		t.Fatalf("key = %q", key)
	}
	_ = os.Unsetenv("TEST_LOOP_API_KEY")
}

func TestResolveAPIModelEnvOverride(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "claude-sonnet-4-6")
	h := model.ADLHarness{Type: "api", Provider: "anthropic", Model: "claude-sonnet-4-20250514"}
	got := resolveAPIModel(RunRequest{Model: h.Model}, h)
	if got != "claude-sonnet-4-6" {
		t.Fatalf("model = %q, want env override", got)
	}
}

func TestResolveAPIModelAgentConfig(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "")
	h := model.ADLHarness{Type: "api", Provider: "anthropic", Model: "default-model"}
	got := resolveAPIModel(RunRequest{
		Model:       h.Model,
		AgentConfig: map[string]any{"model": "session-model"},
	}, h)
	if got != "session-model" {
		t.Fatalf("model = %q", got)
	}
}

func TestResolveAPIModelHarnessDefault(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "")
	h := model.ADLHarness{Type: "api", Provider: "anthropic", Model: "harness-default"}
	got := resolveAPIModel(RunRequest{Model: h.Model}, h)
	if got != "harness-default" {
		t.Fatalf("model = %q", got)
	}
}

func TestResolveAnthropicModelCandidatesCustomGateway(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "http://claudecode.local.dev.example.net:9123")
	t.Setenv("ANTHROPIC_MODEL", "")
	h := model.ADLHarness{Type: "api", Provider: "anthropic", Model: anthropicBuiltinDefaultModel}
	req := RunRequest{Model: h.Model, Env: map[string]string{"ANTHROPIC_BASE_URL": "http://claudecode.local.dev.example.net:9123"}}
	got := resolveAnthropicModelCandidates(req, h, h.Model)
	if len(got) == 0 {
		t.Fatal("expected candidates")
	}
	if got[0] != "claude-sonnet-4-6" {
		t.Fatalf("first candidate = %q, want gateway fallback", got[0])
	}
	foundDefault := false
	for _, m := range got {
		if m == anthropicBuiltinDefaultModel {
			foundDefault = true
		}
	}
	if !foundDefault {
		t.Fatalf("builtin default missing from candidates: %v", got)
	}
}

func TestResolveAnthropicModelCandidatesExplicitEnv(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "http://gateway.example:9123")
	t.Setenv("ANTHROPIC_MODEL", "my-gateway-model")
	h := model.ADLHarness{Type: "api", Provider: "anthropic", Model: anthropicBuiltinDefaultModel}
	req := RunRequest{
		Model: h.Model,
		Env: map[string]string{
			"ANTHROPIC_BASE_URL": "http://gateway.example:9123",
			"ANTHROPIC_MODEL":    "my-gateway-model",
		},
	}
	got := resolveAnthropicModelCandidates(req, h, h.Model)
	if len(got) != 1 || got[0] != "my-gateway-model" {
		t.Fatalf("candidates = %v, want only explicit env model", got)
	}
}

func TestResolveAnthropicModelCandidatesPublicAPI(t *testing.T) {
	t.Setenv("ANTHROPIC_BASE_URL", "")
	t.Setenv("ANTHROPIC_MODEL", "")
	h := model.ADLHarness{Type: "api", Provider: "anthropic", Model: anthropicBuiltinDefaultModel}
	got := resolveAnthropicModelCandidates(RunRequest{Model: h.Model}, h, h.Model)
	if len(got) != 1 || got[0] != anthropicBuiltinDefaultModel {
		t.Fatalf("candidates = %v", got)
	}
}

func TestNormalizeToolCallArguments(t *testing.T) {
	if got := normalizeToolCallArguments(""); got != "{}" {
		t.Fatalf("empty = %q", got)
	}
	if got := normalizeToolCallArguments(`{"html":"<p>x</p>"}`); got != `{"html":"<p>x</p>"}` {
		t.Fatalf("object = %q", got)
	}
	if got := normalizeToolCallArguments("{not json"); got != "{}" {
		t.Fatalf("invalid = %q", got)
	}
}

func TestAccumulatedToolCallCumulativeArgs(t *testing.T) {
	acc := &accumulatedToolCall{id: "call-1", name: "ask_user", started: true}
	chunks := []string{
		`{"title": "C`,
		`{"title": "Chart`,
		`{"title": "Chart Preferences", "message": "Pick one", "questions": [{"question": "Type?"}]}`,
	}
	var deltas []string
	var lastEmitted string
	for _, chunk := range chunks {
		prev := lastEmitted
		lastEmitted = chunk
		acc.args.Reset()
		acc.args.WriteString(chunk)
		if delta, changed := toolArgsStreamUpdate(prev, chunk); changed {
			deltas = append(deltas, delta)
		}
	}
	final := acc.args.String()
	if final != chunks[len(chunks)-1] {
		t.Fatalf("final args = %q, want latest cumulative chunk", final)
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(final), &parsed); err != nil {
		t.Fatalf("final args not valid JSON: %v", err)
	}
	questions, ok := parsed["questions"].([]any)
	if !ok || len(questions) != 1 {
		t.Fatalf("questions = %#v", parsed["questions"])
	}
	lastDelta := deltas[len(deltas)-1]
	if err := json.Unmarshal([]byte(lastDelta), &parsed); err != nil {
		t.Fatalf("final delta should be complete JSON snapshot: %v", err)
	}
}

func TestNormalizeAPIMessagesToolCalls(t *testing.T) {
	msgs := normalizeAPIMessages([]llm.Message{
		{Role: llm.RoleUser, Content: "chart this"},
		{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{{
				ID:   "call_1",
				Type: "function",
				Function: llm.FunctionCall{
					Name:      "show_visualization",
					Arguments: "",
				},
			}},
		},
	})
	if len(msgs) != 2 || len(msgs[1].ToolCalls) != 1 {
		t.Fatalf("messages = %+v", msgs)
	}
	if msgs[1].ToolCalls[0].Function.Arguments != "{}" {
		t.Fatalf("args = %q", msgs[1].ToolCalls[0].Function.Arguments)
	}
}

func TestResolveAPIBaseURLEnv(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://gateway.example/v1")
	h := model.ADLHarness{Type: "api", Provider: "openai"}
	profile := ResolveAPIProviderProfile(h)
	got := resolveAPIBaseURL(profile, h, nil)
	if got != "https://gateway.example/v1" {
		t.Fatalf("base url = %q", got)
	}
}

func TestResolveAPIBaseURLADLOverride(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "https://env.example/v1")
	h := model.ADLHarness{Type: "api", Provider: "openai", BaseURL: "https://adl.example/v1"}
	profile := ResolveAPIProviderProfile(h)
	got := resolveAPIBaseURL(profile, h, nil)
	if got != "https://adl.example/v1" {
		t.Fatalf("base url = %q", got)
	}
}

func TestResolveAPIBaseURLFromMergedEnv(t *testing.T) {
	t.Setenv("OPENAI_BASE_URL", "")
	h := model.ADLHarness{Type: "api", Provider: "openai"}
	profile := ResolveAPIProviderProfile(h)
	got := resolveAPIBaseURL(profile, h, map[string]string{"OPENAI_BASE_URL": "https://adl-env.example/v1"})
	if got != "https://adl-env.example/v1" {
		t.Fatalf("base url = %q", got)
	}
}

type fakeOllamaLister struct {
	models []string
	err    error
}

func (f fakeOllamaLister) Name() string { return "ollama" }

func (f fakeOllamaLister) Completion(context.Context, llm.CompletionParams) (*llm.ChatCompletion, error) {
	return nil, fmt.Errorf("not implemented")
}

func (f fakeOllamaLister) CompletionStream(context.Context, llm.CompletionParams) (<-chan llm.ChatCompletionChunk, <-chan error) {
	ch := make(chan llm.ChatCompletionChunk)
	errCh := make(chan error, 1)
	close(ch)
	errCh <- fmt.Errorf("not implemented")
	close(errCh)
	return ch, errCh
}

func (f fakeOllamaLister) ListModels(context.Context) (*llm.ModelsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := make([]llm.Model, len(f.models))
	for i, id := range f.models {
		out[i] = llm.Model{ID: id}
	}
	return &llm.ModelsResponse{Data: out}, nil
}

func TestPickOllamaModelAutoPick(t *testing.T) {
	got, err := pickOllamaModel("llama3.2", []string{"mistral:latest", "qwen2.5"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "mistral:latest" {
		t.Fatalf("model = %q, want first installed", got)
	}
}

func TestPickOllamaModelMatch(t *testing.T) {
	got, err := pickOllamaModel("llama3.2", []string{"llama3.2:latest"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "llama3.2:latest" {
		t.Fatalf("model = %q", got)
	}
}

func TestEnsureOllamaModelUsesLister(t *testing.T) {
	got, err := ensureOllamaModel(context.Background(), fakeOllamaLister{models: []string{"mistral:latest"}}, "llama3.2")
	if err != nil || got != "mistral:latest" {
		t.Fatalf("got %q err %v", got, err)
	}
}
