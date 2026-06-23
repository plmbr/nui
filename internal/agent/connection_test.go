// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agent

import "testing"

func TestSanitizeConnectionID(t *testing.T) {
	if got := SanitizeConnectionID("ext:corp-pack/echo"); got != "ext-corp-pack-echo" {
		t.Fatalf("got %q", got)
	}
}
