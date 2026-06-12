// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	"embed"
	"io/fs"
	"log"
)

//go:embed ui/dist
var rawUIFiles embed.FS

//go:embed extensions
var rawExtFiles embed.FS

func uiDistFS() fs.FS {
	sub, err := fs.Sub(rawUIFiles, "ui/dist")
	if err != nil {
		log.Fatalf("failed to sub ui/dist: %v", err)
	}
	return sub
}

func extFilesFS() embed.FS {
	return rawExtFiles
}
