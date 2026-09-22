package agent

import (
	"github.com/aldous/jevium/internal/page"
	"testing"
)

func TestOutcomeDistinguishesChangesFromVerification(t *testing.T) {
	a := page.Action{Node: "toggle", Kind: "click", Label: "Enabled", Checked: "false"}
	before := page.WithFingerprint(page.Page{Text: "Before", Actions: []page.Action{a}})
	after := page.WithFingerprint(page.Page{Text: "Clock ticked", Actions: []page.Action{a}})
	if got := ActionOutcome(a, nil, before, after); got != "observed-change" {
		t.Fatalf("unrelated change = %s", got)
	}
	after.Actions[0].Checked = "true"
	after = page.WithFingerprint(after)
	if got := ActionOutcome(a, nil, before, after); got != "verified" {
		t.Fatalf("toggle outcome = %s", got)
	}
	if got := ActionOutcome(a, nil, before, before); got != "unchanged" {
		t.Fatalf("no change = %s", got)
	}
}

func TestStopsRepeatedToggleCycle(t *testing.T) {
	h := []HistoryEntry{}
	for i := 0; i < 3; i++ {
		h = append(h, HistoryEntry{Kind: "click", Action: "Menu", Before: "closed", After: "open"}, HistoryEntry{Kind: "click", Action: "Menu", Before: "open", After: "closed"})
	}
	if !RepeatedCycle(h) {
		t.Fatal("toggle cycle was not stopped")
	}
	h[len(h)-1].After = "new page"
	if RepeatedCycle(h) {
		t.Fatal("new transition mistaken for cycle")
	}
}
