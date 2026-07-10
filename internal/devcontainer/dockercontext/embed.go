// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

// Package dockercontext embeds Dockerfile build contexts for Loop-managed devcontainer images.
// Keep these files in sync with docker/devcontainer-* at the repo root.
package dockercontext

import "embed"

//go:embed devcontainer-claude-code/Dockerfile devcontainer-pi/Dockerfile devcontainer-codex/Dockerfile devcontainer-opencode/Dockerfile
var FS embed.FS
