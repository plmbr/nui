// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"context"
	"testing"

	"loop/internal/llm"
	"loop/internal/model"
)

type mockStreamProvider struct {
	chunks []llm.ChatCompletionChunk
}

func (p *mockStreamProvider) Name() string { return "mock" }

func (p *mockStreamProvider) Completion(context.Context, llm.CompletionParams) (*llm.ChatCompletion, error) {
	return nil, nil
}

func (p *mockStreamProvider) CompletionStream(context.Context, llm.CompletionParams) (<-chan llm.ChatCompletionChunk, <-chan error) {
	chunks := make(chan llm.ChatCompletionChunk, len(p.chunks))
	errs := make(chan error, 1)
	go func() {
		defer close(chunks)
		defer close(errs)
		for _, chunk := range p.chunks {
			chunks <- chunk
		}
	}()
	return chunks, errs
}

func TestStreamCompletion_nativeAskUserBlockedOnOllama(t *testing.T) {
	provider := &mockStreamProvider{
		chunks: []llm.ChatCompletionChunk{{
			Choices: []llm.ChunkChoice{{
				Delta: llm.ChunkDelta{
					ToolCalls: []llm.ToolCall{{
						ID:   "call_0",
						Type: "function",
						Function: llm.FunctionCall{
							Name:      "ask_user",
							Arguments: `{"message":"I can help with many tasks."}`,
						},
					}},
				},
			}},
		}},
	}
	agent := &APIHarnessAgent{Harness: model.ADLHarness{Type: "api", Provider: "ollama"}}
	events := make(chan Event, 4)
	assistant, text, err := agent.streamCompletion(
		context.Background(),
		provider,
		"test",
		nil,
		[]llm.Tool{{Type: "function", Function: llm.Function{Name: "ask_user"}}},
		"what can you do",
		events,
	)
	close(events)
	if err != nil {
		t.Fatal(err)
	}
	if len(assistant.ToolCalls) != 0 {
		t.Fatalf("tool calls = %#v", assistant.ToolCalls)
	}
	if text != "" {
		t.Fatalf("text = %q, want empty so Run can plain-text fallback", text)
	}
	for ev := range events {
		if ev.Type == EventText {
			t.Fatalf("unexpected text event: %q", ev.Content)
		}
	}
}

func TestStreamPlainTextCompletion_emitsText(t *testing.T) {
	provider := &mockStreamProvider{
		chunks: []llm.ChatCompletionChunk{{
			Choices: []llm.ChunkChoice{{
				Delta: llm.ChunkDelta{Content: "Hello!"},
			}},
		}},
	}
	agent := &APIHarnessAgent{Harness: model.ADLHarness{Type: "api", Provider: "ollama"}}
	events := make(chan Event, 2)
	text, err := agent.streamPlainTextCompletion(context.Background(), provider, "test", nil, events)
	close(events)
	if err != nil {
		t.Fatal(err)
	}
	if text != "Hello!" {
		t.Fatalf("text = %q", text)
	}
}
