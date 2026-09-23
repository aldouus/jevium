package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"strings"
)

type fixtureNode = appium.FixtureNode

var screen = appium.FixtureScreen
var bounds = appium.FixtureBounds
var window = appium.FixtureWindow
var scroll = appium.FixtureScroll
var button = appium.FixtureButton
var textField = appium.FixtureTextField
var staticText = appium.FixtureStaticText

type mockElement struct {
	ID   string
	Node fixtureNode
}

func elementResponse(r recorded, elements []mockElement) (any, bool) {
	if len(elements) == 0 {
		return nil, false
	}
	if strings.HasSuffix(r.Path, "/elements") {
		refs := []map[string]string{}
		for _, e := range elements {
			if r.Body["value"] == "XCUIElementType"+string(e.Node.Kind) {
				refs = append(refs, map[string]string{"element-6066-11e4-a52e-4f735466cecf": e.ID})
			}
		}
		return refs, true
	}
	if strings.HasSuffix(r.Path, "/element/active") {
		return map[string]string{"element-6066-11e4-a52e-4f735466cecf": elements[0].ID}, true
	}
	for _, e := range elements {
		if !strings.Contains(r.Path, "/element/"+e.ID+"/") {
			continue
		}
		if strings.HasSuffix(r.Path, "/attribute/name") {
			return e.Node.Name, true
		}
		if strings.HasSuffix(r.Path, "/rect") {
			return map[string]float64{"x": e.Node.Rect.X, "y": e.Node.Rect.Y, "width": e.Node.Rect.W, "height": e.Node.Rect.H}, true
		}
	}
	return nil, false
}
