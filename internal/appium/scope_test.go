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
	source := `<AppiumAUT><XCUIElementTypeWindow width="400" height="800">
 <XCUIElementTypeScrollView width="400" height="800">
  <XCUIElementTypeWebView x="100" y="0" width="300" height="800"/>
 </XCUIElementTypeScrollView></XCUIElementTypeWindow></AppiumAUT>`
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
	source := `<AppiumAUT><XCUIElementTypeWindow width="400" height="800">
 <XCUIElementTypeScrollView x="100" width="300" height="800">
  <XCUIElementTypeWebView x="100" width="300" height="800"/>
 </XCUIElementTypeScrollView>
 <XCUIElementTypeTable width="100" height="800"/>
 </XCUIElementTypeWindow></AppiumAUT>`
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
