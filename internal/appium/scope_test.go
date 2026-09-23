package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"testing"
)

func TestWebViewScopeDoesNotLeakToToolbar(t *testing.T) {
	source := screen(window(375, 812,
		scroll("", bounds(0, 0, 375, 700), fixtureNode{Kind: "WebView", Rect: bounds(0, 0, 375, 700), Children: []fixtureNode{button("Site menu", bounds(0, 0, 60, 40))}}),
		button("Browser menu", bounds(0, 720, 60, 40)),
	))
	p, err := appium.SnapshotFromSource(source, "browser", nil)
	if err != nil {
		t.Fatal(err)
	}
	scopes := map[string]string{}
	for _, a := range p.Actions {
		scopes[a.Label] = a.Scope
	}
	if scopes["Site menu"] != "web" || scopes["Browser menu"] != "native" {
		t.Fatalf("scopes = %v", scopes)
	}
	for _, scope := range []string{"web", "native"} {
		count := 0
		for _, a := range p.Actions {
			if a.Kind == "scroll" && a.Scope == scope {
				count++
			}
		}
		want := 2
		if scope == "native" {
			want = 0
		}
		if count != want {
			t.Fatalf("%s scroll count=%d", scope, count)
		}
	}
}

func TestNativeScrollUsesOnlyExposedPartOfWebWrapper(t *testing.T) {
	source := screen(window(400, 800, scroll("", bounds(0, 0, 400, 800), fixtureNode{Kind: "WebView", Rect: bounds(100, 0, 300, 800)})))
	p, err := appium.SnapshotFromSource(source, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, a := range p.Actions {
		if a.Kind == "scroll" && a.Scope == "native" {
			count++
			if a.Rect == nil || a.Rect.X != 0 || a.Rect.W != 100 || a.Rect.H != 800 {
				t.Fatalf("native gesture overlaps web content: %+v", a.Rect)
			}
		}
	}
	if count != 2 {
		t.Fatalf("native scroll count=%d", count)
	}
}

func TestWebWrapperDoesNotHideSiblingNativeScroll(t *testing.T) {
	source := screen(window(400, 800,
		scroll("", bounds(100, 0, 300, 800), fixtureNode{Kind: "WebView", Rect: bounds(100, 0, 300, 800)}),
		fixtureNode{Kind: "Table", Rect: bounds(0, 0, 100, 800)},
	))
	p, err := appium.SnapshotFromSource(source, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, a := range p.Actions {
		if a.Kind == "scroll" && a.Scope == "native" {
			count++
			if a.Rect == nil || a.Rect.X != 0 || a.Rect.W != 100 {
				t.Fatalf("wrong native region: %+v", a.Rect)
			}
		}
	}
	if count != 2 {
		t.Fatalf("native scroll count=%d", count)
	}
}
