// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package cmd

import (
	"bytes"
	"testing"
)

func TestVersionCmd(t *testing.T) {
	SetVersion("1.2.3-test")
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})
	t.Cleanup(func() {
		rootCmd.SetArgs(nil)
		rootCmd.SetOut(nil)
		rootCmd.SetErr(nil)
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "1.2.3-test\n" {
		t.Fatalf("got %q", got)
	}
}
