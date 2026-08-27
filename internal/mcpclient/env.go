// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpclient

import "nui/internal/store"

func envWithOverrides(overrides map[string]string) []string {
	return store.MergeProcessEnv(overrides)
}
