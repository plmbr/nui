// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agents

import (
	"fmt"
	"strings"

	"nui/internal/appversion"
	"nui/internal/extensions"
	"nui/internal/store"
)

// EnvironmentSnapshot is a compact, non-secret summary of the local nui install.
type EnvironmentSnapshot struct {
	Version          string
	DefaultHarness   string
	Theme            string
	AgentCount       int
	ExtensionCount   int
	DisabledExtCount int
	MCPServerCount   int
	HarnessCount     int
}

// CollectEnvironmentSnapshot gathers counts for the nui system-prompt appendix.
// agentCount should be the number of discoverable agent types (caller-supplied to avoid import cycles).
func CollectEnvironmentSnapshot(settings store.Settings, agentCount int) EnvironmentSnapshot {
	snap := EnvironmentSnapshot{
		Version:        appversion.Get(),
		DefaultHarness: strings.TrimSpace(settings.DefaultHarness),
		Theme:          strings.TrimSpace(settings.Theme),
		AgentCount:     agentCount,
	}
	if snap.Theme == "" {
		snap.Theme = "light"
	}
	if snap.DefaultHarness == "" || !HarnessAvailable(snap.DefaultHarness) {
		snap.DefaultHarness = PickDefaultHarnessRef(settings)
	}

	// Prefer the in-memory registry. extensions.List() calls LoadRegistry() and
	// fully rescans/reinitializes programmatic hosts — too expensive for every
	// OrchestratorDefinition (agent-types list + each orchestrate turn).
	var extInfos []extensions.ExtensionInfo
	if extensions.Default != nil {
		extInfos = extensions.Default.Info()
	} else if listed, err := extensions.List(); err == nil {
		extInfos = listed
	}
	snap.ExtensionCount = len(extInfos)
	for _, e := range extInfos {
		if e.Disabled {
			snap.DisabledExtCount++
		}
	}

	if servers, err := store.LoadMCPServers(); err == nil {
		snap.MCPServerCount = len(servers)
	}

	snap.HarnessCount = len(SelectableHarnessRefs())
	return snap
}

// FormatEnvironmentAppendix renders the dynamic "## Current environment" block.
func FormatEnvironmentAppendix(snap EnvironmentSnapshot) string {
	var b strings.Builder
	b.WriteString("## Current environment\n")
	if v := strings.TrimSpace(snap.Version); v != "" {
		fmt.Fprintf(&b, "- version: %s\n", v)
	}
	fmt.Fprintf(&b, "- defaultHarness: %s\n", orDash(snap.DefaultHarness))
	fmt.Fprintf(&b, "- theme: %s\n", orDash(snap.Theme))
	fmt.Fprintf(&b, "- agents: %d available (call list_agents for details)\n", snap.AgentCount)
	fmt.Fprintf(&b, "- extensions: %d installed", snap.ExtensionCount)
	if snap.DisabledExtCount > 0 {
		fmt.Fprintf(&b, ", %d disabled", snap.DisabledExtCount)
	}
	b.WriteString(" (call list_extensions)\n")
	fmt.Fprintf(&b, "- mcp servers: %d configured (call list_mcp_servers)\n", snap.MCPServerCount)
	fmt.Fprintf(&b, "- harnesses: %d available (call list_harnesses)\n", snap.HarnessCount)
	return strings.TrimSpace(b.String())
}

// ApproximateAgentCount returns a best-effort count of discoverable agent types.
func ApproximateAgentCount() int {
	n := len(BuiltinAgentDefs())
	if userDefs, err := store.LoadADLDefinitions(); err == nil {
		n += len(userDefs)
	}
	if extensions.Default != nil {
		for _, info := range extensions.Default.Info() {
			if info.Disabled {
				continue
			}
			n += len(info.Agents)
			n += len(info.Harnesses) // harness-only agent types
		}
	}
	return n
}

// NuiEnvironmentAppendix builds the appendix from current settings.
// When agentCount < 0, ApproximateAgentCount is used.
func NuiEnvironmentAppendix(settings store.Settings, agentCount int) string {
	if agentCount < 0 {
		agentCount = ApproximateAgentCount()
	}
	return FormatEnvironmentAppendix(CollectEnvironmentSnapshot(settings, agentCount))
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}
