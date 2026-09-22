package agent

import (
	"encoding/json"
	"github.com/aldous/jevium/internal/page"
	"os"
	"path/filepath"
	"testing"
)

func TestCoverageSeparatesObservedAttemptedAndVerified(t *testing.T) {
	action := page.Action{Node: "field", Kind: "fill", Label: "Name"}
	p := page.WithFingerprint(page.Page{URL: "https://example.test/", Actions: []page.Action{action}})
	a := &Agent{State: State{Page: p, Status: "ready"}}
	path := filepath.Join(t.TempDir(), "coverage.json")
	if err := a.EnableCoverage(path, []string{"https://example.test/unvisited"}); err != nil {
		t.Fatal(err)
	}
	a.Coverage.Attempt(p, action)
	if err := a.saveCoverage(); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report Coverage
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	got := report.Pages[p.URL].Controls[controlKey(action)]
	if got.Attempts != 1 || got.ObservedResults != 0 || got.Verified != 0 {
		t.Fatalf("pending result = %+v", got)
	}
	if report.Pages["https://example.test/unvisited"].Visited || report.CompleteInventory {
		t.Fatal("claimed unobserved coverage")
	}
	text := "Ada"
	next := p
	next.Actions = []page.Action{action}
	next.Actions[0].Value = text
	next = page.WithFingerprint(next)
	a.Coverage.Result(p, action, next, &text)
	got = a.Coverage.Pages[p.URL].Controls[controlKey(action)]
	if got.ObservedResults != 1 || got.Changed != 1 || got.Verified != 1 {
		t.Fatalf("verified result = %+v", got)
	}
}
