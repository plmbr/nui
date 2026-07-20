// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package model

import "testing"

func TestNormalizeADLTags(t *testing.T) {
	def := ADLDefinition{Tags: []string{" coding ", "research", "coding", "", "  "}}
	NormalizeADLTags(&def)
	if len(def.Tags) != 2 || def.Tags[0] != "coding" || def.Tags[1] != "research" {
		t.Fatalf("got %#v", def.Tags)
	}
}

func TestValidateADLTags(t *testing.T) {
	if err := ValidateADLTags([]string{"coding", "research"}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateADLTags([]string{"coding", "coding"}); err == nil {
		t.Fatal("expected duplicate tag error")
	}
	if err := ValidateADLTags([]string{""}); err == nil {
		t.Fatal("expected empty tag error")
	}
}
