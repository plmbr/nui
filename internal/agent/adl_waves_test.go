// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"

	"nui/internal/model"
)

func TestTopoWavesParallelIndependent(t *testing.T) {
	steps := []model.ADLStep{
		{Name: "a"},
		{Name: "b"},
		{Name: "c", DependsOn: []string{"a", "b"}},
	}
	waves, err := topoWaves(steps)
	if err != nil {
		t.Fatal(err)
	}
	if len(waves) != 2 {
		t.Fatalf("waves = %d, want 2", len(waves))
	}
	if len(waves[0]) != 2 {
		t.Fatalf("first wave size = %d, want 2", len(waves[0]))
	}
	if len(waves[1]) != 1 || waves[1][0].Name != "c" {
		t.Fatalf("second wave = %+v", waves[1])
	}
}

func TestExpandBuiltinNuiMCPRefs(t *testing.T) {
	out := expandBuiltinNuiMCPRefs([]model.ADLMCPServer{
		{Name: "nui-council", Ref: "builtin:nui"},
		{Name: "other", Command: "echo"},
	})
	if len(out) != 2 {
		t.Fatalf("len = %d", len(out))
	}
	if out[0].Ref != "" || out[0].Command == "" {
		t.Fatalf("expected resolved builtin:nui, got %+v", out[0])
	}
	if len(out[0].Args) != 1 || out[0].Args[0] != "mcp" {
		t.Fatalf("args = %v", out[0].Args)
	}
	if out[1].Command != "echo" {
		t.Fatalf("other = %+v", out[1])
	}
}
