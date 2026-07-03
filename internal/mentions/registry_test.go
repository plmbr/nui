// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mentions_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/mentions"
)

func TestBuiltinFilesProviderRoot(t *testing.T) {
	p := mentions.BuiltinFilesProvider{}
	resp, err := p.List(context.Background(), mentions.ListRequest{Parent: ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 1 || resp.Items[0].Value != mentions.BuiltinFilesRoot {
		t.Fatalf("root items: %+v", resp.Items)
	}
	if !resp.Items[0].HasChildren {
		t.Fatal("expected Files & folders to have children")
	}
}

func TestBuiltinFilesProviderListAndResolve(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"alpha.go", "beta.txt", "docs"} {
		path := filepath.Join(dir, name)
		if strings.HasSuffix(name, "docs") {
			if err := os.Mkdir(path, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	p := mentions.BuiltinFilesProvider{}
	resp, err := p.List(context.Background(), mentions.ListRequest{
		WorkingDir: dir,
		Parent:     mentions.BuiltinFilesRoot,
		Limit:      mentions.MaxFileItems,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != 3 {
		t.Fatalf("items: %+v", resp.Items)
	}

	var fileValue, dirValue string
	for _, item := range resp.Items {
		if item.Value == "file:alpha.go" {
			fileValue = item.Value
		}
		if item.Value == "dir:docs" {
			dirValue = item.Value
			if item.HasChildren {
				t.Fatalf("docs should be selectable: %+v", item)
			}
		}
	}
	if fileValue == "" {
		t.Fatal("alpha.go not listed")
	}
	if dirValue == "" {
		t.Fatal("docs not listed")
	}

	resolved, err := p.Resolve(context.Background(), mentions.ResolveRequest{
		WorkingDir: dir,
		Value:      fileValue,
	})
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(dir, "alpha.go")
	if resolved != "@"+want {
		t.Fatalf("file resolved = %q, want %q", resolved, "@"+want)
	}

	dirResolved, err := p.Resolve(context.Background(), mentions.ResolveRequest{
		WorkingDir: dir,
		Value:      dirValue,
	})
	if err != nil {
		t.Fatal(err)
	}
	wantDir := filepath.Join(dir, "docs")
	if dirResolved != "@"+wantDir {
		t.Fatalf("dir resolved = %q, want %q", dirResolved, "@"+wantDir)
	}
}

func TestBuiltinFilesProviderBFSNested(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "pkg", "util"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg", "util", "helper.go"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	p := mentions.BuiltinFilesProvider{}
	resp, err := p.List(context.Background(), mentions.ListRequest{
		WorkingDir: dir,
		Parent:     mentions.BuiltinFilesRoot,
		Limit:      mentions.MaxFileItems,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range resp.Items {
		if item.Value == "file:pkg/util/helper.go" {
			found = true
		}
	}
	if !found {
		t.Fatalf("nested file not found: %+v", resp.Items)
	}
}

func TestBuiltinFilesProviderFilterAndLimit(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 120; i++ {
		name := fmt.Sprintf("match-%03d.txt", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	p := mentions.BuiltinFilesProvider{}
	resp, err := p.List(context.Background(), mentions.ListRequest{
		WorkingDir: dir,
		Parent:     mentions.BuiltinFilesRoot,
		Query:      "match",
		Limit:      mentions.MaxFileItems,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) != mentions.MaxFileItems {
		t.Fatalf("got %d items, want %d", len(resp.Items), mentions.MaxFileItems)
	}
}

func TestRegistryResolveMessage(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(filePath, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	dirPath := filepath.Join(dir, "docs")
	if err := os.Mkdir(dirPath, 0o755); err != nil {
		t.Fatal(err)
	}

	reg := mentions.NewRegistry(nil)
	msg := "please read @file:hello.txt in @dir:docs now"
	got, err := reg.ResolveMessage(context.Background(), dir, msg, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "please read @" + filePath + " in @" + dirPath + " now"
	if got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

func TestRegistryResolveMessageAbsoluteUploadPath(t *testing.T) {
	uploadDir := filepath.Join(os.TempDir(), "loop-uploads", "sess-1")
	if err := os.MkdirAll(uploadDir, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Join(os.TempDir(), "loop-uploads", "sess-1")) })

	imagePath := filepath.Join(uploadDir, "photo.png")
	if err := os.WriteFile(imagePath, []byte{0x89, 0x50, 0x4e, 0x47}, 0o600); err != nil {
		t.Fatal(err)
	}

	reg := mentions.NewRegistry(nil)
	msg := "look at @" + imagePath
	got, err := reg.ResolveMessage(context.Background(), t.TempDir(), msg, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := "look at @" + imagePath
	if got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}
