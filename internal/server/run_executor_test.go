// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"nui/internal/model"
)

func TestPrepareRunMessage_expandsSlashSkill(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", filepath.Join(tmp, "home"))

	session := model.Session{AgentType: "claude-code"}
	got, err := prepareRunMessage(context.Background(), session, "/tmp/work", "/create-agent save as helper", false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(strings.TrimSpace(got), "/create-agent") {
		t.Fatalf("expected expanded skill body, got %q", got)
	}
	if !strings.Contains(got, "Create Agent") {
		t.Fatalf("expected create-agent skill body, got %q", got)
	}
	if !strings.Contains(got, "save as helper") {
		t.Fatalf("expected trailing user args, got %q", got)
	}
}
