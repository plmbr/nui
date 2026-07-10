// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import (
	"testing"
)

func TestHarnessRunners_includesDevcontainer(t *testing.T) {
	if _, ok := harnessRunners["devcontainer"]; !ok {
		t.Fatal("harnessRunners missing devcontainer entry")
	}
}
