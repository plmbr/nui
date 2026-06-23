// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

// IsADLWorkingDirInput reports whether the user chooses the session working directory.
// When false (default), Loop provisions an isolated workspace under ~/.loop/workspaces/<session-id>.
func IsADLWorkingDirInput(def ADLDefinition) bool {
	return def.WorkingDirInput
}
