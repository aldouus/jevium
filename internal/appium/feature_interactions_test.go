package appium

import (
	"testing"

	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
)

func TestScopedDraggingDoesNotCrossContentBoundary(t *testing.T) {
	web := page.Action{Kind: "click", Node: "web", Scope: "web", Role: "button", Label: "Site", Rect: &page.Rect{W: 40, H: 40}}
	native := page.Action{Kind: "click", Node: "native", Scope: "native", Role: "button", Label: "Toolbar", Rect: &page.Rect{Y: 100, W: 40, H: 40}}
	actions, _ := gestureActions([]page.Action{web, native}, map[any]bool{"web": true, "native": true})
	p := page.Page{Actions: actions}
	drags := 0
	for _, action := range policy.ScopedActions(p, "all") {
		if action.Kind == "drag" {
			drags++
		}
	}
	if drags != 2 {
		t.Fatalf("all scope has %d drags, want both directions", drags)
	}
	for _, scope := range []string{"web", "native"} {
		for _, action := range policy.ScopedActions(p, scope) {
			if action.Kind == "drag" {
				t.Fatalf("%s scope allows cross-boundary drag: %+v", scope, action)
			}
		}
	}
}
