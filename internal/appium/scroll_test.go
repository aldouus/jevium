package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/policy"
	"testing"
)

func TestNestedScrollTargetsExposeSeparateGestureRegions(t *testing.T) {
	const source = `<AppiumAUT>
  <XCUIElementTypeWindow width="400" height="800">
    <XCUIElementTypeScrollView name="Page" x="0" y="0" width="400" height="800">
      <XCUIElementTypeScrollView name="Results" x="100" y="100" width="300" height="600"/>
    </XCUIElementTypeScrollView>
  </XCUIElementTypeWindow>
</AppiumAUT>`
	p, err := appium.SnapshotFromSource(source, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, targets, controls := policy.ActionSpace(p.Actions)
	if len(targets["SCROLL_DOWN"]) != 2 || len(targets["SCROLL_UP"]) != 2 {
		t.Fatalf("targets=%v", targets)
	}
	if _, exists := controls["SCROLL_DOWN"]; exists {
		t.Fatal("scroll must choose a target")
	}
	parent, child := targets["SCROLL_DOWN"]["1"], targets["SCROLL_DOWN"]["2"]
	if parent.Rect == nil || child.Rect == nil {
		t.Fatal("missing scroll rectangles")
	}
	if parent.Rect.X != 0 || parent.Rect.W != 100 || parent.Rect.H != 800 {
		t.Fatalf("parent region=%+v", parent.Rect)
	}
	if child.Rect.X != 100 || child.Rect.Y != 100 || child.Rect.W != 300 || child.Rect.H != 600 {
		t.Fatalf("child region=%+v", child.Rect)
	}
}

func TestParentKeepsUsableStripAfterSeveralChildren(t *testing.T) {
	const source = `<AppiumAUT>
  <XCUIElementTypeWindow width="400" height="800">
    <XCUIElementTypeScrollView name="Page" x="0" y="0" width="400" height="800">
      <XCUIElementTypeScrollView name="Top" x="100" y="0" width="200" height="100"/>
      <XCUIElementTypeScrollView name="Bottom" x="0" y="100" width="400" height="700"/>
    </XCUIElementTypeScrollView>
  </XCUIElementTypeWindow>
</AppiumAUT>`
	p, err := appium.SnapshotFromSource(source, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, targets, _ := policy.ActionSpace(p.Actions)
	if len(targets["SCROLL_DOWN"]) != 3 {
		t.Fatalf("targets=%v", targets)
	}
	parent := targets["SCROLL_DOWN"]["1"]
	if parent.Rect == nil || parent.Rect.X != 0 || parent.Rect.Y != 0 || parent.Rect.W != 100 || parent.Rect.H != 100 {
		t.Fatalf("parent=%+v", parent.Rect)
	}
}
