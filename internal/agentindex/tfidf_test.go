// Copyright (c) Mehmet Bektas <mbektasgh@outlook.com>

package agentindex

import "testing"

func TestCosine(t *testing.T) {
	if got := Cosine([]float64{1, 0}, []float64{1, 0}); got < 0.999 {
		t.Fatalf("identical = %v", got)
	}
	if got := Cosine([]float64{1, 0}, []float64{0, 1}); got > 0.001 {
		t.Fatalf("orthogonal = %v", got)
	}
}

func TestBuildDocHashStable(t *testing.T) {
	a := BuildDoc("acme-search", "Acme Search", "Find Acme docs", []string{"search"})
	b := BuildDoc("acme-search", "Acme Search", "Find Acme docs", []string{"search"})
	if a.Hash == "" || a.Hash != b.Hash {
		t.Fatalf("hash mismatch %q vs %q", a.Hash, b.Hash)
	}
	c := BuildDoc("acme-search", "Acme Search", "Find Acme docs!", []string{"search"})
	if a.Hash == c.Hash {
		t.Fatal("expected hash to change")
	}
}

func TestScorePrefersMatchingDoc(t *testing.T) {
	docs := []Doc{
		BuildDoc("docs-finder", "Acme Docs Finder", "Looks up internal documentation and library references", []string{"docs", "lookup"}),
		BuildDoc("calendar", "Acme Calendar", "Schedules meetings on the calendar", []string{"calendar"}),
	}
	scores := Score("find the library documentation reference", docs)
	if scores["docs-finder"] <= scores["calendar"] {
		t.Fatalf("scores = %#v, want docs-finder ahead", scores)
	}
	if scores["docs-finder"] < 0.2 {
		t.Fatalf("docs-finder score too low: %v", scores["docs-finder"])
	}
}

func TestScoreEmptyQuery(t *testing.T) {
	docs := []Doc{BuildDoc("a", "Alpha", "alpha helper", nil)}
	scores := Score("", docs)
	if len(scores) != 0 && scores["a"] != 0 {
		t.Fatalf("scores = %#v", scores)
	}
}

func TestSemanticBonus(t *testing.T) {
	if got := SemanticBonus(0.805); got != 81 {
		t.Fatalf("got %d, want 81", got)
	}
	if got := SemanticBonus(0.1); got != 0 {
		t.Fatalf("below min got %d", got)
	}
	if got := SemanticBonus(-1); got != 0 {
		t.Fatalf("got %d", got)
	}
}
