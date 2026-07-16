// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package main

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionRaw string

func version() string {
	return strings.TrimSpace(versionRaw)
}
