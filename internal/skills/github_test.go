// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import "testing"

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		in           string
		wantClone    string
		wantPath     string
		wantRef      string
		wantOK       bool
	}{
		{
			in:        "https://github.com/example/agent-skills",
			wantClone: "https://github.com/example/agent-skills.git",
			wantOK:    true,
		},
		{
			in:        "https://github.com/example/agent-skills.git",
			wantClone: "https://github.com/example/agent-skills.git",
			wantOK:    true,
		},
		{
			in:        "github.com/example/agent-skills/tree/main/skills/code-review",
			wantClone: "https://github.com/example/agent-skills.git",
			wantPath:  "skills/code-review",
			wantRef:   "main",
			wantOK:    true,
		},
		{
			in:        "https://github.com/example/agent-skills/tree/v1.0.0/skills/shared-style",
			wantClone: "https://github.com/example/agent-skills.git",
			wantPath:  "skills/shared-style",
			wantRef:   "v1.0.0",
			wantOK:    true,
		},
		{
			in:        "https://github.com/example/agent-skills/blob/main/skills/code-review/SKILL.md",
			wantClone: "https://github.com/example/agent-skills.git",
			wantPath:  "skills/code-review",
			wantRef:   "main",
			wantOK:    true,
		},
		{
			in:        "git@github.com:example/agent-skills.git",
			wantClone: "https://github.com/example/agent-skills.git",
			wantOK:    true,
		},
		{
			in:        "https://github.example.com/example/agents/blob/main/watchdog.yaml",
			wantClone: "https://github.example.com/example/agents.git",
			wantPath:  "watchdog.yaml",
			wantRef:   "main",
			wantOK:    true,
		},
		{
			in:        "https://github.example.com/example/agents.git",
			wantClone: "https://github.example.com/example/agents.git",
			wantOK:    true,
		},
		{
			in:        "git@github.example.com:example/agents.git",
			wantClone: "https://github.example.com/example/agents.git",
			wantOK:    true,
		},
		{
			in:        "github.example.com/example/agents/tree/main/agents",
			wantClone: "https://github.example.com/example/agents.git",
			wantPath:  "agents",
			wantRef:   "main",
			wantOK:    true,
		},
		{
			in:     "./local/path",
			wantOK: false,
		},
		{
			in:     "https://gitlab.com/example/repo",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			clone, path, ref, ok := ParseGitHubURL(tt.in)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if clone != tt.wantClone {
				t.Errorf("cloneURL = %q, want %q", clone, tt.wantClone)
			}
			if path != tt.wantPath {
				t.Errorf("repoPath = %q, want %q", path, tt.wantPath)
			}
			if ref != tt.wantRef {
				t.Errorf("ref = %q, want %q", ref, tt.wantRef)
			}
		})
	}
}

func TestNormalizeRepoSkillPath(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"skills/foo", "skills/foo"},
		{"skills/foo/SKILL.md", "skills/foo"},
		{"/skills/foo/SKILL.md/", "skills/foo"},
		{"SKILL.md", "."},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeRepoSkillPath(tt.in); got != tt.want {
			t.Errorf("normalizeRepoSkillPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsGitRemote(t *testing.T) {
	if !IsGitRemote("https://github.com/example/repo/tree/main/skills/foo") {
		t.Fatal("expected github tree URL to be git remote")
	}
	if !IsGitRemote("git@github.com:example/repo.git") {
		t.Fatal("expected ssh github URL to be git remote")
	}
	if !IsGitRemote("https://github.example.com/example/agents/blob/main/watchdog.yaml") {
		t.Fatal("expected GitHub Enterprise blob URL to be git remote")
	}
	if IsGitRemote("./skills/foo") {
		t.Fatal("expected local path not to be git remote")
	}
	if IsGitRemote("/abs/path/skills/foo") {
		t.Fatal("expected absolute local path not to be git remote")
	}
}
