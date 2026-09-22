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
	p := page.Page{Text: "Saved", Actions: []page.Action{{Node: "toggle", Label: "Enabled", Checked: "false"}}}
	if ExpectationsMatch(es, p) {
		t.Fatal("unchecked control satisfied completion")
	}
	p.Actions[0].Checked = "true"
	if !ExpectationsMatch(es, p) {
		t.Fatal("all observed conditions should match")
	}
	p.Actions = append(p.Actions, page.Action{Node: "other", Label: "Enabled", Checked: "true"})
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
