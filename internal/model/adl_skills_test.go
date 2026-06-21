// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import (
	"testing"

	"gopkg.in/yaml.v3"
)

func TestADLDefinitionYAML_skills(t *testing.T) {
	raw := []byte(`adl: "1.0"
id: skills-agent
name: Skills Agent
aiAssets:
  skills:
    - name: review
      path: /tmp/review-skill
    - name: greeter
      content: |
        ---
        name: greeter
        description: Demo greeter
        ---
        Say hello briefly.
`)

	var def ADLDefinition
	if err := yaml.Unmarshal(raw, &def); err != nil {
		t.Fatal(err)
	}
	if len(def.AIAssets.Skills) != 2 {
		t.Fatalf("skills: %v", def.AIAssets.Skills)
	}
	if err := ValidateADLSkills(def.AIAssets.Skills); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeADLSkills_legacy(t *testing.T) {
	def := ADLDefinition{Skill: "/skills/code-review"}
	NormalizeADLSkills(&def)
	if len(def.AIAssets.Skills) != 1 {
		t.Fatalf("skills: %v", def.AIAssets.Skills)
	}
	if def.AIAssets.Skills[0].Name != "code-review" {
		t.Fatalf("name = %q", def.AIAssets.Skills[0].Name)
	}
}

func TestSkillSourceKind_gitRequiresPath(t *testing.T) {
	_, err := SkillSourceKind(ADLSkill{Name: "x", Git: "https://example.com/repo.git"})
	if err == nil {
		t.Fatal("expected error for git without path")
	}
}
