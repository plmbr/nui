// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package mentions_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"nui/internal/mentions"
)

func writeFiles(t *testing.T, dir string, files ...string) {
	t.Helper()
	for _, name := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func resolve(t *testing.T, dir, msg string) string {
	t.Helper()
	got, err := mentions.NewRegistry(nil).ResolveMessage(context.Background(), dir, msg, nil)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestResolveMessageKeepsTrailingPunctuationOutOfThePath(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "data/README.md")
	abs := filepath.Join(dir, "data", "README.md")
	for _, punctuation := range []string{",", ".", ";", ":", "!", "?", ")", "),", ".)", "\u2026", ".\u201d", "\u3002"} {
		msg := "Summarize @file:data/README.md" + punctuation + " then chart it"
		want := "Summarize @" + abs + punctuation + " then chart it"
		if got := resolve(t, dir, msg); got != want {
			t.Errorf("%q: resolved = %q, want %q", punctuation, got, want)
		}
	}
}

func TestResolveMessageTrimsPunctuationOneCharacterAtATime(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "data(2)"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := "compare @" + filepath.Join(dir, "data(2)") + ")."
	if got := resolve(t, dir, "compare @dir:data(2))."); got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

func TestResolveMessagePrefersTheNameAsWritten(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "notes.", "notes")
	want := "read @" + filepath.Join(dir, "notes.") + " first"
	if got := resolve(t, dir, "read @file:notes. first"); got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

func TestResolveMessageNeverReachesADifferentFile(t *testing.T) {
	// Each of these names a file that does not resolve; a looser trim would
	// reach its shorter sibling instead.
	dir := t.TempDir()
	writeFiles(t, dir, "foo.c", "data")
	if err := os.Mkdir(filepath.Join(dir, "data."), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, msg := range []string{
		"open @file:foo.c++ now",       // `+` is part of the name
		"open @file:data. please",      // `data.` exists (a folder), so `data` is never tried
		"open @file:x.md...... please", // more punctuation than is ever trimmed
	} {
		if got := resolve(t, dir, msg); got != msg {
			t.Errorf("resolved = %q, want it unchanged", got)
		}
	}
}

func TestResolveMessageTrimmingStaysInsideTheWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "work")
	writeFiles(t, root, "outside.md", "work/inside.md")
	msg := "see @file:../outside.md, and @file:../outside.md."
	if got := resolve(t, dir, msg); got != msg {
		t.Fatalf("resolved = %q, want it unchanged", got)
	}
}

func TestResolveMessageQuotedPaths(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "data/my notes.md", "img/logo@2x.png", "notes.md")
	spaced := filepath.Join(dir, "data", "my notes.md")
	at := filepath.Join(dir, "img", "logo@2x.png")
	msg := `use @file:"data/my notes.md", @file:"img/logo@2x.png". and @file:"notes.md,"`
	// A resolved path with whitespace is quoted the way Claude Code reads it.
	// Quoting means "exactly this name", so `notes.md,` is not trimmed.
	want := `use @"` + spaced + `", @` + at + `. and @file:"notes.md,"`
	if got := resolve(t, dir, msg); got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

func TestResolveMessageOnlyAWellFormedQuoteStartsAQuotedMention(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "a", "a b.md")
	for _, msg := range []string{
		`read @file:"a"b.csv now`,    // closing quote glued to a word: one plain token
		"read @file:\"a\u2028b.md\"", // a line break cannot sit inside quotes
		`read @file:"a b.md`,         // never closed
		`read @file:"" now`,          // empty
	} {
		if got := resolve(t, dir, msg); got != msg {
			t.Errorf("resolved = %q, want it unchanged", got)
		}
	}
}

func TestResolveMessageMentionMustFollowWhitespace(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "a.md")
	abs := filepath.Join(dir, "a.md")
	for msg, want := range map[string]string{
		"see\u3000@file:a.md": "see\u3000@" + abs,
		"see (@file:a.md)":    "see (@file:a.md)",
		"mail me@file:a.md":   "mail me@file:a.md",
	} {
		if got := resolve(t, dir, msg); got != want {
			t.Errorf("%q: resolved = %q, want %q", msg, got, want)
		}
	}
}

func TestResolveMessageQuotedValueIsExact(t *testing.T) {
	dir := t.TempDir()
	writeFiles(t, dir, "notes.md ", "notes.md")
	want := `read @"` + filepath.Join(dir, "notes.md ") + `" now`
	if got := resolve(t, dir, `read @file:"notes.md " now`); got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}

type fakeExtensions struct{ resolved []string }

func (f *fakeExtensions) ListExtensionRoots() []mentions.Item { return nil }
func (f *fakeExtensions) ListExtension(context.Context, string, string, mentions.ListRequest) (mentions.ListResponse, error) {
	return mentions.ListResponse{}, nil
}
func (f *fakeExtensions) ResolveExtension(_ context.Context, _, _ string, req mentions.ResolveRequest) (string, error) {
	f.resolved = append(f.resolved, req.Value)
	return "[" + req.Value + "]", nil
}
func (f *fakeExtensions) MatchExtensionValue(value string) (string, string, bool) {
	return "demo", "catalog", len(value) > len("ext:demo:catalog:") && value[:len("ext:demo:catalog:")] == "ext:demo:catalog:"
}
func (f *fakeExtensions) MatchExtensionParent(string) (string, string, bool) { return "", "", false }

func TestResolveMessageExtensionValuesAreQuotedButNeverTrimmed(t *testing.T) {
	// Provider values are opaque, so trailing punctuation belongs to them.
	ext := &fakeExtensions{}
	reg := mentions.NewRegistry(ext)
	allowed := map[string]bool{"ext:demo:catalog": true}
	msg := `join @ext:"demo:catalog:Q3 orders" with @ext:demo:catalog:refunds, today`
	got, err := reg.ResolveMessage(context.Background(), t.TempDir(), msg, allowed)
	if err != nil {
		t.Fatal(err)
	}
	want := "join [ext:demo:catalog:Q3 orders] with [ext:demo:catalog:refunds,] today"
	if got != want {
		t.Fatalf("resolved = %q, want %q", got, want)
	}
}
