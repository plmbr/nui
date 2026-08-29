// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package appversion

import "strings"

// Version is the running product version (no "v" prefix). Set at process start
// from the embedded VERSION file or build ldflags.
var Version = "dev"

// Set records the running version.
func Set(v string) {
	v = strings.TrimSpace(v)
	if v == "" {
		v = "dev"
	}
	Version = v
}

// Get returns the running version.
func Get() string {
	return Version
}
