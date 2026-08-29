// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import "strings"

// mergeSettings overlays user preferences onto system defaults.
// Non-empty user scalars and non-nil user pointers/slices/maps win.
func mergeSettings(sys, user Settings) Settings {
	out := sys
	normalizeSettings(&out)

	if user.Theme != "" {
		out.Theme = user.Theme
	}
	if user.UITheme != "" {
		out.UITheme = user.UITheme
	}
	if user.DisableSloganAnimation != nil {
		out.DisableSloganAnimation = user.DisableSloganAnimation
	}
	if user.DefaultAgentType != "" {
		out.DefaultAgentType = user.DefaultAgentType
	}
	if user.DefaultHarness != "" {
		out.DefaultHarness = user.DefaultHarness
	}
	if user.DisabledExtensions != nil {
		out.DisabledExtensions = append([]string(nil), user.DisabledExtensions...)
	}
	if strings.TrimSpace(user.MCPOAuthCallbackURL) != "" {
		out.MCPOAuthCallbackURL = user.MCPOAuthCallbackURL
	}
	if user.MemoryUserMode != "" {
		out.MemoryUserMode = user.MemoryUserMode
	}
	if user.MemoryAgentsMode != nil {
		out.MemoryAgentsMode = make(map[string]string, len(sys.MemoryAgentsMode)+len(user.MemoryAgentsMode))
		for k, v := range sys.MemoryAgentsMode {
			out.MemoryAgentsMode[k] = v
		}
		for k, v := range user.MemoryAgentsMode {
			out.MemoryAgentsMode[k] = v
		}
	}
	if user.AutoCheckUpdates != nil {
		out.AutoCheckUpdates = user.AutoCheckUpdates
	}
	if user.UpdateCheckIntervalHours > 0 {
		out.UpdateCheckIntervalHours = user.UpdateCheckIntervalHours
	}
	if user.SkippedUpdateVersion != "" {
		out.SkippedUpdateVersion = user.SkippedUpdateVersion
	}
	normalizeSettings(&out)
	return out
}

// mergeEnvMaps overlays user env keys onto system (user wins per key).
func mergeEnvMaps(sys, user map[string]string) map[string]string {
	out := make(map[string]string, len(sys)+len(user))
	for k, v := range sys {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	for k, v := range user {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}
