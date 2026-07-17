// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"nui/internal/model"
)

// mergeADLEnv combines top-level ADL env with harness-level env (harness wins on conflict).
func mergeADLEnv(def model.ADLDefinition, harness model.ADLHarness) map[string]string {
	merged := make(map[string]string, len(def.Env)+len(harness.Env))
	for k, v := range def.Env {
		merged[k] = v
	}
	for k, v := range harness.Env {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}
