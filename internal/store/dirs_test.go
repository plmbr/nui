// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtraConfigDirs(t *testing.T) {
	t.Setenv(envExtraConfigDirs, "")
	if got := ExtraConfigDirs(); len(got) != 0 {
		t.Fatalf("empty env: %+v", got)
	}

	sep := string(os.PathListSeparator)
	t.Setenv(envExtraConfigDirs, "/first"+sep+" /second "+sep+sep+"")
	got := ExtraConfigDirs()
	if len(got) != 2 || got[0] != "/first" || got[1] != "/second" {
		t.Fatalf("dirs = %+v", got)
	}
}

func TestAppendExtraConfigDir(t *testing.T) {
	t.Setenv(envExtraConfigDirs, "")
	AppendExtraConfigDir("/a")
	AppendExtraConfigDir("/b")
	got := ExtraConfigDirs()
	if len(got) != 2 || got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("dirs = %+v", got)
	}
}

func TestAgentAndExtensionConfigDirs(t *testing.T) {
	home := t.TempDir()
	extra := filepath.Join(t.TempDir(), "extra")
	system := filepath.Join(t.TempDir(), "system")
	if err := os.MkdirAll(filepath.Join(extra, "agents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(extra, "extensions", "pack-a"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(system, "agents"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".nui", "agents"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(home, ".nui", "extensions"), 0700); err != nil {
		t.Fatal(err)
	}

	t.Setenv("HOME", home)
	t.Setenv(envExtraConfigDirs, extra)
	t.Setenv(envSystemConfig, system)

	agentDirs, err := AgentConfigDirs()
	if err != nil {
		t.Fatal(err)
	}
	wantAgents := []string{
		filepath.Join(extra, "agents"),
		filepath.Join(system, "agents"),
		filepath.Join(home, ".nui", "agents"),
	}
	if len(agentDirs) != len(wantAgents) {
		t.Fatalf("agent dirs = %+v", agentDirs)
	}
	for i, want := range wantAgents {
		if agentDirs[i] != want {
			t.Fatalf("agentDirs[%d] = %q want %q", i, agentDirs[i], want)
		}
	}

	extDirs, err := ExtensionConfigDirs()
	if err != nil {
		t.Fatal(err)
	}
	wantExt := []string{
		filepath.Join(extra, "extensions"),
		filepath.Join(home, ".nui", "extensions"),
	}
	if len(extDirs) != len(wantExt) {
		t.Fatalf("extension dirs = %+v", extDirs)
	}
	for i, want := range wantExt {
		if extDirs[i] != want {
			t.Fatalf("extDirs[%d] = %q want %q", i, extDirs[i], want)
		}
	}
}
