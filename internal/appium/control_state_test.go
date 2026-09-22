package appium_test

import (
	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/appium"
	"testing"
)

func TestPickerCurrentStateWithoutOfferedOptions(t *testing.T) {
	p, err := appium.SnapshotFromSource(`<AppiumAUT>
 <XCUIElementTypePickerWheel name="Country" label="Country" value="Canada" width="100" height="100"/>
 <XCUIElementTypeTextField label="Unknown" y="100" width="100" height="40"/>
 </AppiumAUT>`, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !agent.ExpectationsMatch([]agent.Expectation{{Field: "value", Label: "Country", Value: "Canada"}}, p) {
		t.Fatal("observed current picker value missed")
	}
	if agent.ExpectationsMatch([]agent.Expectation{{Field: "value", Label: "Unknown", Value: ""}}, p) {
		t.Fatal("missing native value mistaken for empty")
	}
	for _, a := range p.Actions {
		if a.Kind == "select" {
			t.Fatal("invented option action")
		}
	}
}
