// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

// Package ui embeds the Vite production build (ui/dist) for serving from Go.
package ui

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed all:dist
var rawDist embed.FS

// DistFS returns the embedded ui/dist filesystem (index.html at root).
func DistFS() fs.FS {
	sub, err := fs.Sub(rawDist, "dist")
	if err != nil {
		log.Fatalf("failed to sub ui/dist: %v", err)
	}
	return sub
}
