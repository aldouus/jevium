package appium_test

import (
	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/appium"
	"testing"
)

func TestMissingSwitchValueCannotProveUnchecked(t *testing.T) {
	source := screen(fixtureNode{Kind: "Switch", Label: "Enabled", Rect: bounds(0, 0, 100, 40)})
	p, err := appium.SnapshotFromSource(source, "settings", nil)
	if err != nil {
		t.Fatal(err)
	}
	if agent.ExpectationsMatch([]agent.Expectation{{Field: "checked", Label: "Enabled", Value: "false"}}, p) {
		t.Fatal("unknown switch state satisfied unchecked expectation")
	}
}
