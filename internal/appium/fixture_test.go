package appium_test

import (
	"encoding/xml"
	"strconv"
	"strings"

	"github.com/aldous/jevium/internal/page"
)

type fixtureKind string

type fixtureNode struct {
	Kind                                      fixtureKind
	Name, Label, Value, BundleID, Values      string
	Rect                                      page.Rect
	Hidden, Disabled, NotHittable, Accessible bool
	AnimationTick                             string
	Children                                  []fixtureNode
	ValuePresent                              bool
}

func (n fixtureNode) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	start := xml.StartElement{Name: xml.Name{Local: "XCUIElementType" + string(n.Kind)}}
	attr := func(name, value string) {
		if value != "" {
			start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: name}, Value: value})
		}
	}
	attr("name", n.Name)
	attr("label", n.Label)
	attr("value", n.Value)
	if n.ValuePresent && n.Value == "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "value"}, Value: ""})
	}
	attr("bundleId", n.BundleID)
	attr("values", n.Values)
	attr("animationTick", n.AnimationTick)
	attr("x", strconv.FormatFloat(n.Rect.X, 'f', -1, 64))
	attr("y", strconv.FormatFloat(n.Rect.Y, 'f', -1, 64))
	attr("width", strconv.FormatFloat(n.Rect.W, 'f', -1, 64))
	attr("height", strconv.FormatFloat(n.Rect.H, 'f', -1, 64))
	if n.Hidden {
		attr("visible", "false")
	}
	if n.Disabled {
		attr("enabled", "false")
	}
	if n.NotHittable {
		attr("hittable", "false")
	}
	if n.Accessible {
		attr("accessible", "true")
	}
	if err := e.EncodeToken(start); err != nil {
		return err
	}
	for _, child := range n.Children {
		if err := e.Encode(child); err != nil {
			return err
		}
	}
	return e.EncodeToken(start.End())
}

func screen(children ...fixtureNode) string {
	root := struct {
		XMLName  xml.Name `xml:"AppiumAUT"`
		Children []fixtureNode
	}{Children: children}
	raw, err := xml.Marshal(root)
	if err != nil {
		panic(err)
	}
	return string(raw)
}

func bounds(x, y, w, h float64) page.Rect { return page.Rect{X: x, Y: y, W: w, H: h} }
func window(w, h float64, children ...fixtureNode) fixtureNode {
	return fixtureNode{Kind: "Window", Rect: bounds(0, 0, w, h), Children: children}
}
func scroll(name string, r page.Rect, children ...fixtureNode) fixtureNode {
	return fixtureNode{Kind: "ScrollView", Name: name, Rect: r, Children: children}
}
func button(name string, r page.Rect) fixtureNode {
	return fixtureNode{Kind: "Button", Name: name, Label: name, Rect: r}
}
func textField(name, value string, r page.Rect) fixtureNode {
	return fixtureNode{Kind: "TextField", Name: name, Label: name, Value: value, ValuePresent: true, Rect: r}
}
func staticText(label string, r page.Rect) fixtureNode {
	return fixtureNode{Kind: "StaticText", Label: label, Rect: r}
}

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
