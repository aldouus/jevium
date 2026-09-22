package policy

import (
	"github.com/aldous/jevium/internal/page"
	"testing"
)

func TestScopeRestrictsChoices(t *testing.T) {
	p := page.Page{URL: "app://browser", Actions: []page.Action{
		{ID: "site", Kind: "click", Scope: "web"}, {ID: "toolbar", Kind: "click", Scope: "native"}, {ID: "wait", Kind: "wait"},
	}}
	got := ScopedActions(p, "web")
	if len(got) != 2 || got[0].ID != "site" || got[1].ID != "wait" {
		t.Fatalf("web choices = %+v", got)
	}
	if len(ScopedActions(p, "all")) != 3 {
		t.Fatal("all scope lost controls")
	}
	native := ScopedActions(page.Page{URL: "app://settings", Actions: []page.Action{{ID: "setting", Kind: "click"}}}, "native")
	if len(native) != 1 || native[0].Scope != "native" {
		t.Fatalf("native choices = %+v", native)
	}
}
