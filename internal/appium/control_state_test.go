package appium_test

import (
	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/appium"
	"testing"
)

func TestPickerCurrentStateWithoutOfferedOptions(t *testing.T) {
	source := screen(fixtureNode{Kind: "PickerWheel", Name: "Country", Label: "Country", Value: "Canada", Rect: bounds(0, 0, 100, 100)}, fixtureNode{Kind: "TextField", Label: "Unknown", Rect: bounds(0, 100, 100, 40)})
	p, err := appium.SnapshotFromSource(source, "app", nil)
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
