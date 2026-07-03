// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package skills

import "loop/internal/model"

// AgentSkills returns all skills available to an agent definition, including builtins.
func AgentSkills(def model.ADLDefinition) []model.ADLSkill {
	defCopy := def
	model.NormalizeADLSkills(&defCopy)

	var skills []model.ADLSkill
	skills = append(skills, defCopy.AIAssets.Skills...)
	for _, step := range defCopy.Steps {
		skills = append(skills, step.AIAssets.Skills...)
	}
	return WithBuiltins(skills)
}
