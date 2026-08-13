// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import (
	"net/url"
	"path"
	"strings"
)

// ParseGitHubURL parses a GitHub or GitHub Enterprise web/git URL into a clone
// URL, repo-relative skill path, and ref (branch/tag/commit from tree/blob URLs).
// ok is false when the input is not a GitHub repository URL.
func ParseGitHubURL(raw string) (cloneURL, repoPath, ref string, ok bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", "", "", false
	}
	if !strings.Contains(raw, "://") && !strings.HasPrefix(raw, "git@") {
		if host, rest, cutOK := strings.Cut(raw, "/"); cutOK && isGitHubHost(host) {
			raw = "https://" + host + "/" + rest
		}
	}

	if strings.HasPrefix(raw, "git@") {
		rest := strings.TrimPrefix(raw, "git@")
		host, pathPart, cutOK := strings.Cut(rest, ":")
		if !cutOK || !isGitHubHost(host) {
			return "", "", "", false
		}
		pathPart = strings.TrimSuffix(pathPart, ".git")
		parts := strings.SplitN(pathPart, "/", 2)
		if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
			return "", "", "", false
		}
		repo := strings.TrimSuffix(parts[1], ".git")
		return githubCloneURL(host, parts[0], repo), "", "", true
	}

	u, err := url.Parse(raw)
	if err != nil || !isGitHubHost(u.Hostname()) {
		return "", "", "", false
	}

	segments := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(segments) < 2 {
		return "", "", "", false
	}
	owner := segments[0]
	repo := strings.TrimSuffix(segments[1], ".git")
	cloneURL = githubCloneURL(u.Hostname(), owner, repo)

	if len(segments) >= 4 && (segments[2] == "tree" || segments[2] == "blob") {
		ref = segments[3]
		if len(segments) > 4 {
			repoPath = normalizeRepoSkillPath(path.Join(segments[4:]...))
		}
		return cloneURL, repoPath, ref, true
	}

	return cloneURL, "", "", true
}

func isGitHubHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimPrefix(host, "www.")
	return host == "github.com" || strings.HasPrefix(host, "github.")
}

func githubCloneURL(host, owner, repo string) string {
	host = strings.TrimPrefix(strings.ToLower(host), "www.")
	return "https://" + host + "/" + owner + "/" + repo + ".git"
}

// IsGitRemote reports whether s looks like a git remote URL rather than a local path.
func IsGitRemote(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "git@") {
		return true
	}
	if strings.HasSuffix(s, ".git") {
		return true
	}
	if strings.Contains(s, "://") {
		if _, _, _, ok := ParseGitHubURL(s); ok {
			return true
		}
		lower := strings.ToLower(s)
		return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
	}
	if host, rest, cutOK := strings.Cut(s, "/"); cutOK && rest != "" && isGitHubHost(host) {
		return true
	}
	return false
}

// normalizeRepoSkillPath returns the skill directory path within a repo.
// When repoPath points at SKILL.md, the parent directory is returned.
func normalizeRepoSkillPath(repoPath string) string {
	repoPath = strings.TrimSpace(repoPath)
	repoPath = strings.Trim(repoPath, "/")
	if repoPath == "" {
		return ""
	}
	if strings.EqualFold(path.Base(repoPath), skillFileName) {
		repoPath = path.Dir(repoPath)
	}
	return strings.Trim(repoPath, "/")
}
