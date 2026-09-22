package appium_test

import (
	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/appium"
	"testing"
)

func TestMissingSwitchValueCannotProveUnchecked(t *testing.T) {
	p, err := appium.SnapshotFromSource(`<AppiumAUT>
 <XCUIElementTypeSwitch label="Enabled" x="0" y="0" width="100" height="40"/>
 </AppiumAUT>`, "settings", nil)
	if err != nil {
		t.Fatal(err)
	}
	if agent.ExpectationsMatch([]agent.Expectation{{Field: "checked", Label: "Enabled", Value: "false"}}, p) {
		t.Fatal("unknown switch state satisfied unchecked expectation")
	}
}
