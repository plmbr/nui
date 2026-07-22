// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mcpclient

import (
	"os"
	"strings"
)

func envWithOverrides(overrides map[string]string) []string {
	if len(overrides) == 0 {
		return os.Environ()
	}
	m := make(map[string]string)
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			m[k] = v
		}
	}
	for k, v := range overrides {
		m[k] = v
	}
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}
