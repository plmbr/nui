// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package update

import (
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

const (
	defaultRepo      = "plmbr/nui"
	defaultUserAgent = "nui-updater"
)

// Kind selects which release asset family to use.
type Kind string

const (
	KindCLI     Kind = "cli"
	KindDesktop Kind = "desktop"
)

// Config controls GitHub release lookups and downloads.
type Config struct {
	Repo           string // owner/repo; default plmbr/nui
	CurrentVersion string // no required "v" prefix
	Kind           Kind   // cli or desktop
	GOOS           string // default runtime.GOOS
	GOARCH         string // default runtime.GOARCH
	HTTPClient     *http.Client
	UserAgent      string
	Token          string // optional GitHub token (env GITHUB_TOKEN)
}

func (c Config) withDefaults() Config {
	if c.Repo == "" {
		c.Repo = strings.TrimSpace(os.Getenv("NUI_UPDATE_REPO"))
	}
	if c.Repo == "" {
		c.Repo = defaultRepo
	}
	if c.Kind == "" {
		c.Kind = KindCLI
	}
	if c.GOOS == "" {
		c.GOOS = runtime.GOOS
	}
	if c.GOARCH == "" {
		c.GOARCH = runtime.GOARCH
	}
	if c.UserAgent == "" {
		c.UserAgent = defaultUserAgent
	}
	if c.Token == "" {
		c.Token = strings.TrimSpace(os.Getenv("GITHUB_TOKEN"))
	}
	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{Timeout: 60 * time.Second}
	}
	return c
}

// PlatformArch maps Go arch names to release asset arch labels.
func PlatformArch(goarch string) string {
	switch goarch {
	case "amd64", "arm64":
		return goarch
	default:
		return goarch
	}
}
