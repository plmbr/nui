// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"encoding/json"
	"testing"
)

func TestExtractImageBlockMCPFormat(t *testing.T) {
	block := map[string]any{
		"type":     "image",
		"data":     "aGVsbG8=",
		"mimeType": "image/png",
	}
	img, ok := extractImageBlock(block)
	if !ok {
		t.Fatal("expected image")
	}
	if img.Data != "aGVsbG8=" {
		t.Fatalf("data = %q", img.Data)
	}
	if img.MediaType != "image/png" {
		t.Fatalf("media type = %q", img.MediaType)
	}
}

func TestExtractImageBlockAnthropicFormat(t *testing.T) {
	block := map[string]any{
		"type": "image",
		"source": map[string]any{
			"type":       "base64",
			"media_type": "image/jpeg",
			"data":       "abc123",
		},
	}
	img, ok := extractImageBlock(block)
	if !ok {
		t.Fatal("expected image")
	}
	if img.Data != "abc123" || img.MediaType != "image/jpeg" {
		t.Fatalf("unexpected image: %+v", img)
	}
}

func TestExtractImagesFromJSONToolUseResult(t *testing.T) {
	raw := json.RawMessage(`{"content":[{"type":"image","data":"xyz","mimeType":"image/png"}]}`)
	imgs := extractImagesFromJSON(raw)
	if len(imgs) != 1 {
		t.Fatalf("len = %d", len(imgs))
	}
	if imgs[0].Data != "xyz" {
		t.Fatalf("data = %q", imgs[0].Data)
	}
}
