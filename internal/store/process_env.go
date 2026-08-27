// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package store

import (
	"os"
	"strings"
)

// reservedEnvKeys are nui-injected or internal runtime keys that must not be
// set via Customize → Env vars / per-extension env. Extension-owned names such
// as NUI_MY_EXTENSION_TOKEN are allowed.
var reservedEnvKeys = map[string]struct{}{
	"NUI_API_URL":             {},
	"NUI_URL":                 {},
	"NUI_EXTENSION_DIR":       {},
	"NUI_EXTENSION_NAME":      {},
	"NUI_EXTENSION_ENTRY":     {},
	"NUI_HARNESS_ID":          {},
	"NUI_CONNECTION_ID":       {},
	"NUI_SESSION_ID":          {},
	"NUI_RUN_ID":              {},
	"NUI_HITL_SDK_DIR":        {},
	"NUI_HITL_CHANNEL_ID":     {},
	"NUI_MENTION_SDK_DIR":     {},
	"NUI_MENTION_PROVIDER_ID": {},
	"NUI_STORAGE_HANDLER_ID":  {},
	"NUI_MEMORY_AGENT_ID":     {},
	"NUI_MEMORY_USER_MODE":    {},
	"NUI_MEMORY_AGENT_MODE":   {},
	"NUI_MCP_TOOLS_PATH":      {},
	"NUI_MCP_BINARY":          {},
	"NUI_BWRAP_PATH":          {},
	"NUI_CODEX_PATH":          {},
	"NUI_PYTHON3_PATH":        {},
	"NUI_PORT":                {},
	"NUI_HOME":                {},
	"NUI_INSTALL_DIR":         {},
}

// IsReservedEnvKey reports whether key is reserved for nui internals.
// Empty keys are reserved. Only known core NUI_* names are blocked so
// extensions may use their own NUI_-prefixed config keys.
func IsReservedEnvKey(key string) bool {
	key = strings.TrimSpace(key)
	if key == "" {
		return true
	}
	_, ok := reservedEnvKeys[key]
	return ok
}

// AllSecretEnv returns a copy of ~/.nui/secrets.json env (managed + custom).
func AllSecretEnv() map[string]string {
	s, err := LoadSecrets()
	if err != nil || len(s.Env) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(s.Env))
	for k, v := range s.Env {
		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)
		if k == "" || v == "" {
			continue
		}
		out[k] = v
	}
	return out
}

// ApplyGlobalEnvToProcess sets secrets.json values into the current process
// only when the key is unset or blank. Existing process env always wins.
func ApplyGlobalEnvToProcess() {
	for k, v := range AllSecretEnv() {
		if strings.TrimSpace(os.Getenv(k)) != "" {
			continue
		}
		_ = os.Setenv(k, v)
	}
}

// MergeProcessEnv builds a child-process environment:
// launch os.Environ → fill blanks from secrets.json → apply overrides (later wins).
func MergeProcessEnv(overrides ...map[string]string) []string {
	m := environMap()
	for k, v := range AllSecretEnv() {
		if strings.TrimSpace(m[k]) == "" {
			m[k] = v
		}
	}
	for _, o := range overrides {
		for k, v := range o {
			k = strings.TrimSpace(k)
			if k == "" {
				continue
			}
			m[k] = v
		}
	}
	return mapToEnviron(m)
}

// ExtensionProcessEnv merges global secrets, per-extension env, then extras.
// Precedence (later wins): process → secrets (blanks) → per-extension → extras.
func ExtensionProcessEnv(extName string, extras ...map[string]string) []string {
	overrides := make([]map[string]string, 0, 1+len(extras))
	if per := ExtensionEnv(extName); len(per) > 0 {
		overrides = append(overrides, per)
	}
	overrides = append(overrides, extras...)
	return MergeProcessEnv(overrides...)
}

func environMap() map[string]string {
	m := make(map[string]string)
	for _, kv := range os.Environ() {
		k, v, ok := strings.Cut(kv, "=")
		if ok {
			m[k] = v
		}
	}
	return m
}

func mapToEnviron(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	return out
}
