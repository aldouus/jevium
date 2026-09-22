package agent

import (
	"github.com/aldous/jevium/internal/page"
	"testing"
)

func TestCompletionRequiresEveryObservedCondition(t *testing.T) {
	es, err := ParseExpectations([]string{`{"field":"text","value":"Saved"}`, `{"field":"checked","label":"Enabled","value":"true"}`})
	if err != nil {
		t.Fatal(err)
	}
	p := page.Page{Text: "Saved", Controls: []page.Control{{Node: "toggle", Label: "Enabled", Checked: "false"}}}
	if ExpectationsMatch(es, p) {
		t.Fatal("unchecked control satisfied completion")
	}
	p.Controls[0].Checked = "true"
	if !ExpectationsMatch(es, p) {
		t.Fatal("all observed conditions should match")
	}
	p.Controls = append(p.Controls, page.Control{Node: "other", Label: "Enabled", Checked: "true"})
	if ExpectationsMatch(es, p) {
		t.Fatal("ambiguous label satisfied completion")
	}
}

func TestCompletionDoesNotAcceptLastWord(t *testing.T) {
	if GoalVisible("Stop when payment failed is visible", page.Page{Text: "Upload failed"}) {
		t.Fatal("unrelated failure satisfied goal")
	}
	if !GoalVisible("Stop when payment failed is visible", page.Page{Text: "Payment failed"}) {
		t.Fatal("full phrase should match")
	}
}

func TestRejectInvalidExpectations(t *testing.T) {
	for _, raw := range []string{`{"field":"script","value":"true"}`, `{"field":"checked","value":"true"}`, `{"field":"expanded","label":"Menu","value":"yes"}`, `{"field":"text","value":"ok","extra":1}`} {
		if _, err := ParseExpectations([]string{raw}); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if ExpectationsMatch(nil, page.Page{Text: "done"}) {
		t.Fatal("missing criteria proved completion")
	}
}

func TestValueExpectationUsesCurrentControlValue(t *testing.T) {
	es := []Expectation{{Field: "value", Label: "Country", Value: "ca"}}
	value := "fr"
	p := page.Page{Controls: []page.Control{{Node: "country", Label: "Country", Value: &value}}, Actions: []page.Action{{Node: "country", Kind: "select", Label: "Country → Canada", Value: "ca", CurrentValue: "fr"}}}
	if ExpectationsMatch(es, p) {
		t.Fatal("offered option mistaken for current value")
	}
	value = "ca"
	p.Actions = nil
	if !ExpectationsMatch(es, p) {
		t.Fatal("current select value did not match")
	}
	es = []Expectation{{Field: "value", Label: "Search", Value: ""}}
	p.Controls = []page.Control{{Node: "search", Label: "Search"}}
	if ExpectationsMatch(es, p) {
		t.Fatal("button mistaken for empty editable field")
	}
}

func TestSelectedExpectationAcceptsObservedARIABoolean(t *testing.T) {
	es := []Expectation{{Field: "selected", Label: "Details", Value: "true"}}
	for _, value := range []any{true, "true"} {
		p := page.Page{Controls: []page.Control{{Node: 7, Label: "Details", Selected: value}}}
		if !ExpectationsMatch(es, p) {
			t.Fatalf("observed selected=%v should match", value)
		}
	}
	for _, value := range []any{false, "false", "", nil, "mixed"} {
		p := page.Page{Controls: []page.Control{{Node: 7, Label: "Details", Selected: value}}}
		if ExpectationsMatch(es, p) {
			t.Fatalf("selected=%v falsely matched", value)
		}
	}
}
