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

func TestNativeSelectedStateAndSliderValueAreObservable(t *testing.T) {
	tab := fixtureNode{Kind: "Tab", Label: "Profile", Selected: "false", Rect: bounds(0, 0, 80, 40)}
	slider := fixtureNode{Kind: "Slider", Label: "Volume", Value: "50%", Rect: bounds(0, 50, 200, 40)}
	before, err := appium.SnapshotFromSource(screen(tab, slider), "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !agent.ExpectationsMatch([]agent.Expectation{{Field: "selected", Label: "Profile", Value: "false"}, {Field: "value", Label: "Volume", Value: "50%"}}, before) {
		t.Fatal("observed tab selection and slider value did not satisfy completion")
	}
	tab.Selected = "true"
	after, err := appium.SnapshotFromSource(screen(tab, slider), "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if before.Fingerprint == after.Fingerprint || !agent.ExpectationsMatch([]agent.Expectation{{Field: "selected", Label: "Profile", Value: "true"}}, after) {
		t.Fatal("selection change was not observed")
	}
	tab.Selected = ""
	unknown, err := appium.SnapshotFromSource(screen(tab), "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if agent.ExpectationsMatch([]agent.Expectation{{Field: "selected", Label: "Profile", Value: "false"}}, unknown) {
		t.Fatal("missing selection mistaken for false")
	}
}
