// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package devcontainer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"loop/internal/devcontainer/dockercontext"
)

func TestResolveImage(t *testing.T) {
	got, err := ResolveImage("claude-code", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != DefaultImages["claude-code"] {
		t.Fatalf("ResolveImage() = %q", got)
	}

	got, err = ResolveImage("claude-code", "custom:tag")
	if err != nil {
		t.Fatal(err)
	}
	if got != "custom:tag" {
		t.Fatalf("ResolveImage(override) = %q", got)
	}
}

func TestIsLoopManagedImage(t *testing.T) {
	defaultImage := DefaultImages["claude-code"]
	if !IsLoopManagedImage("claude-code", defaultImage) {
		t.Fatal("expected default image to be loop-managed")
	}
	if IsLoopManagedImage("claude-code", "custom:tag") {
		t.Fatal("expected custom image not to be loop-managed")
	}
}

func TestFindRepoBuildContext(t *testing.T) {
	dir, err := findRepoBuildContext("claude-code")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "Dockerfile")); err != nil {
		t.Fatalf("Dockerfile missing in %q: %v", dir, err)
	}
}

func TestMaterializeEmbeddedBuildContext(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir, err := materializeEmbeddedBuildContext("pi")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "pi-coding-agent") {
		t.Fatalf("unexpected Dockerfile: %s", data)
	}

	dir2, err := materializeEmbeddedBuildContext("pi")
	if err != nil {
		t.Fatal(err)
	}
	if dir2 != dir {
		t.Fatalf("dir = %q, want cached %q", dir2, dir)
	}
}

func TestEmbeddedDockerfilesMatchRepo(t *testing.T) {
	repoRoot, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for inner, contextName := range dockerContextDirs {
		embedded, err := dockercontext.FS.ReadFile(filepath.Join(contextName, "Dockerfile"))
		if err != nil {
			t.Fatalf("%q embedded Dockerfile: %v", inner, err)
		}
		onDisk, err := os.ReadFile(filepath.Join(repoRoot, "docker", contextName, "Dockerfile"))
		if err != nil {
			t.Fatalf("%q repo Dockerfile: %v", inner, err)
		}
		if string(embedded) != string(onDisk) {
			t.Fatalf("%q: embedded Dockerfile out of sync with docker/%s/Dockerfile", inner, contextName)
		}
	}
}
