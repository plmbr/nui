// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package server

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"nui/internal/mentions"
)

func TestAGUIMentionResolution(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "note.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := mentions.NewRegistry(nil)
	msg := "read @file:note.txt please"
	got, err := reg.ResolveMessage(context.Background(), dir, msg, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "read @" + filePath + " please"
	if got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}
