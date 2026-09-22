package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"testing"
)

func TestWebViewScopeDoesNotLeakToToolbar(t *testing.T) {
	source := `<AppiumAUT>
 <XCUIElementTypeWindow width="375" height="812">
 <XCUIElementTypeScrollView width="375" height="700">
  <XCUIElementTypeWebView width="375" height="700">
   <XCUIElementTypeButton label="Site menu" width="60" height="40"/>
  </XCUIElementTypeWebView>
 </XCUIElementTypeScrollView>
  <XCUIElementTypeButton label="Browser menu" y="720" width="60" height="40"/>
 </XCUIElementTypeWindow>
 </AppiumAUT>`
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
		if count != 2 {
			t.Fatalf("%s scroll count=%d", scope, count)
		}
	}
}
