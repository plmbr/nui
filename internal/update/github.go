// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ReleaseInfo describes a GitHub release asset suitable for this platform.
type ReleaseInfo struct {
	Tag         string `json:"tag"`
	Version     string `json:"version"`
	Name        string `json:"name,omitempty"`
	Notes       string `json:"notes,omitempty"`
	AssetName   string `json:"assetName"`
	AssetURL    string `json:"assetURL"`
	Size        int64  `json:"size"`
	ChecksumURL string `json:"checksumURL,omitempty"`
}

type ghRelease struct {
	TagName    string    `json:"tag_name"`
	Name       string    `json:"name"`
	Body       string    `json:"body"`
	Prerelease bool      `json:"prerelease"`
	Assets     []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type ghAPIError struct {
	Message string `json:"message"`
}

func githubAPIError(status string, body []byte) error {
	msg := parseGitHubErrorMessage(body)
	if msg != "" {
		return fmt.Errorf("github releases: %s: %s", status, msg)
	}
	return fmt.Errorf("github releases: %s", status)
}

func parseGitHubErrorMessage(body []byte) string {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return ""
	}
	var parsed ghAPIError
	if err := json.Unmarshal(body, &parsed); err == nil {
		if msg := strings.TrimSpace(parsed.Message); msg != "" {
			return msg
		}
	}
	if len(trimmed) > 160 {
		return trimmed[:157] + "..."
	}
	return trimmed
}

// Check fetches the latest (or tagged) release and resolves the asset for cfg.
// newer is true when the release version is strictly newer than cfg.CurrentVersion
// (or when force-compare is not needed — caller decides). Use IsNewer for comparison.
func Check(ctx context.Context, cfg Config) (ReleaseInfo, error) {
	cfg = cfg.withDefaults()
	rel, err := fetchLatestRelease(ctx, cfg)
	if err != nil {
		return ReleaseInfo{}, err
	}
	info, err := pickAsset(rel, cfg)
	if err != nil {
		return ReleaseInfo{}, err
	}
	return info, nil
}

// CheckTag fetches a specific release tag.
func CheckTag(ctx context.Context, cfg Config, tag string) (ReleaseInfo, error) {
	cfg = cfg.withDefaults()
	tag = CanonicalTag(tag)
	if tag == "" {
		return ReleaseInfo{}, fmt.Errorf("empty release tag")
	}
	rel, err := fetchReleaseByTag(ctx, cfg, tag)
	if err != nil {
		return ReleaseInfo{}, err
	}
	return pickAsset(rel, cfg)
}

func fetchLatestRelease(ctx context.Context, cfg Config) (ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", cfg.Repo)
	return fetchReleaseURL(ctx, cfg, url)
}

func fetchReleaseByTag(ctx context.Context, cfg Config, tag string) (ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", cfg.Repo, tag)
	return fetchReleaseURL(ctx, cfg, url)
}

func fetchReleaseURL(ctx context.Context, cfg Config, url string) (ghRelease, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return ghRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", cfg.UserAgent)
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	res, err := cfg.HTTPClient.Do(req)
	if err != nil {
		return ghRelease{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return ghRelease{}, err
	}
	if res.StatusCode != http.StatusOK {
		return ghRelease{}, githubAPIError(res.Status, body)
	}
	var rel ghRelease
	if err := json.Unmarshal(body, &rel); err != nil {
		return ghRelease{}, fmt.Errorf("decode release: %w", err)
	}
	if strings.TrimSpace(rel.TagName) == "" {
		return ghRelease{}, fmt.Errorf("release missing tag_name")
	}
	return rel, nil
}

func pickAsset(rel ghRelease, cfg Config) (ReleaseInfo, error) {
	osName := cfg.GOOS
	arch := PlatformArch(cfg.GOARCH)
	tag := rel.TagName
	prefix := "nui_"
	if cfg.Kind == KindDesktop {
		prefix = "nui-desktop_"
	}
	wantPrefix := fmt.Sprintf("%s%s_%s_%s.", prefix, tag, osName, arch)
	// Also accept without trailing dot variants via HasPrefix of stem.
	stem := fmt.Sprintf("%s%s_%s_%s", prefix, tag, osName, arch)

	var chosen *ghAsset
	var checksumURL string
	for i := range rel.Assets {
		a := &rel.Assets[i]
		if a.Name == "checksums.txt" {
			checksumURL = a.BrowserDownloadURL
			continue
		}
		if strings.HasPrefix(a.Name, stem+".") || a.Name == stem+".tar.gz" || a.Name == stem+".zip" {
			chosen = a
			continue
		}
		// Fallback: prefix match for slightly different naming.
		if strings.HasPrefix(a.Name, wantPrefix) {
			chosen = a
		}
	}
	if chosen == nil {
		return ReleaseInfo{}, fmt.Errorf("no %s asset for %s/%s in release %s", cfg.Kind, osName, arch, tag)
	}
	if checksumURL == "" {
		checksumURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", cfg.Repo, tag)
	}
	return ReleaseInfo{
		Tag:         tag,
		Version:     NormalizeVersion(tag),
		Name:        rel.Name,
		Notes:       rel.Body,
		AssetName:   chosen.Name,
		AssetURL:    chosen.BrowserDownloadURL,
		Size:        chosen.Size,
		ChecksumURL: checksumURL,
	}, nil
}
