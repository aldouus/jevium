package appium

import (
	"encoding/xml"
	"io"
	"strconv"
	"strings"

	"github.com/aldous/jevium/internal/page"
)

var clickTypes = map[string]struct{}{
	"XCUIElementTypeButton": {}, "XCUIElementTypeIcon": {}, "XCUIElementTypeCell": {},
	"XCUIElementTypeLink": {}, "XCUIElementTypeTab": {}, "XCUIElementTypeImage": {},
	"XCUIElementTypeKey": {}, "XCUIElementTypeMenuItem": {}, "XCUIElementTypeCollectionViewCell": {},
}
var fillTypes = map[string]struct{}{
	"XCUIElementTypeTextField": {}, "XCUIElementTypeSearchField": {}, "XCUIElementTypeTextView": {},
}
var selectTypes = map[string]struct{}{"XCUIElementTypePickerWheel": {}}
var toggleTypes = map[string]struct{}{"XCUIElementTypeSwitch": {}, "XCUIElementTypeCheckBox": {}}
var skipTypes = map[string]struct{}{"XCUIElementTypeSecureTextField": {}}
var textTypes = map[string]struct{}{"XCUIElementTypeStaticText": {}, "XCUIElementTypeTextView": {}}

var roles = map[string]string{
	"XCUIElementTypeButton": "button", "XCUIElementTypeIcon": "button", "XCUIElementTypeCell": "button",
	"XCUIElementTypeLink": "link", "XCUIElementTypeTab": "tab", "XCUIElementTypeImage": "button",
	"XCUIElementTypeKey": "button", "XCUIElementTypeMenuItem": "menuitem",
	"XCUIElementTypeCollectionViewCell": "button", "XCUIElementTypeTextField": "textbox",
	"XCUIElementTypeSearchField": "searchbox", "XCUIElementTypeTextView": "textbox",
	"XCUIElementTypePickerWheel": "combobox", "XCUIElementTypeSwitch": "switch",
	"XCUIElementTypeCheckBox": "checkbox",
}

type node struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Nodes   []node     `xml:",any"`
}

func (n node) attr(names ...string) string {
	for _, want := range names {
		for _, a := range n.Attrs {
			if a.Name.Local == want && a.Value != "" {
				return a.Value
			}
		}
	}
	return ""
}

func (n node) boolAttr(name string, def bool) bool {
	v := n.attr(name)
	if v == "" {
		return def
	}
	return v != "false" && v != "0" && strings.ToLower(v) != "no"
}

func (n node) intAttr(name string) int {
	v := n.attr(name)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0
	}
	return int(f)
}

func (n node) local() string { return n.XMLName.Local }

func (n node) visible() bool { return n.boolAttr("visible", true) && n.boolAttr("enabled", true) }

func (n node) label() string {
	if v := n.attr("label", "name", "value"); v != "" {
		return v
	}
	return n.local()
}

func (n node) rect() page.Rect {
	return page.Rect{X: float64(n.intAttr("x")), Y: float64(n.intAttr("y")), W: float64(n.intAttr("width")), H: float64(n.intAttr("height"))}
}

func onScreen(r page.Rect, window *page.Rect) bool {
	if r.W <= 0 || r.H <= 0 {
		return false
	}
	if window == nil {
		return true
	}
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	return cx >= window.X && cy >= window.Y && cx < window.X+window.W && cy < window.Y+window.H
}

func intersect(a page.Rect, b *page.Rect) page.Rect {
	if b == nil {
		return a
	}
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	return page.Rect{X: x, Y: y, W: max(0, min(a.X+a.W, b.X+b.W)-x), H: max(0, min(a.Y+a.H, b.Y+b.H)-y)}
}

func scrollContainer(kind string) bool {
	return kind == "XCUIElementTypeScrollView" || kind == "XCUIElementTypeWebView" || kind == "XCUIElementTypeTable" || kind == "XCUIElementTypeCollectionView"
}

func walk(n node, fn func(node)) {
	fn(n)
	for _, c := range n.Nodes {
		walk(c, fn)
	}
}

func SnapshotFromSource(source, bundleID string, window *page.Rect) (page.Page, error) {
	dec := xml.NewDecoder(strings.NewReader(source))
	dec.CharsetReader = func(_ string, input io.Reader) (io.Reader, error) { return input, nil }
	var root node
	if err := dec.Decode(&root); err != nil {
		return page.Page{}, err
	}
	var app node
	foundApp := false
	walk(root, func(n node) {
		if n.local() == "XCUIElementTypeApplication" && !foundApp {
			app, foundApp = n, true
		}
	})
	if !foundApp {
		app = root
	}
	if window == nil {
		walk(root, func(n node) {
			if n.local() == "XCUIElementTypeWindow" {
				w, h := n.intAttr("width"), n.intAttr("height")
				if w > 0 && h > 0 && window == nil {
					r := n.rect()
					window = &r
				}
			}
		})
	}
	appName := app.attr("name", "label")
	if appName == "" {
		parts := strings.Split(bundleID, ".")
		appName = parts[len(parts)-1]
	}
	appBundle := app.attr("bundleId")
	if appBundle == "" {
		appBundle = bundleID
	}
	hasAlert := false
	walk(root, func(n node) {
		if n.local() == "XCUIElementTypeAlert" && n.visible() {
			hasAlert = true
		}
	})
	actions := []page.Action{}
	words := []string{}
	var scrollRect *page.Rect
	nextID := 1
	identity := map[*node]string{}
	var assign func(*node) string
	assign = func(n *node) string {
		if id, ok := identity[n]; ok {
			return id
		}
		id := n.local() + ":" + n.attr("name") + ":" + strconv.Itoa(nextID)
		nextID++
		identity[n] = id
		return id
	}
	var visit func(*node, *page.Rect)
	visit = func(n *node, clip *page.Rect) {
		kind := n.local()
		r := n.rect()
		if (scrollContainer(kind) || kind == "XCUIElementTypeWindow") && r.W > 0 && r.H > 0 {
			bounds := intersect(r, clip)
			clip = &bounds
			if scrollContainer(kind) && n.visible() && n.boolAttr("hittable", true) && bounds.W > 0 && bounds.H > 0 && scrollRect == nil {
				scrollRect = &bounds
			}
		}
		visiblePart := intersect(r, clip)
		if _, ok := textTypes[kind]; ok && n.visible() && visiblePart.W > 0 && visiblePart.H > 0 {
			if t := n.attr("value", "label", "name"); t != "" {
				words = append(words, t)
			}
		}
		actionable := has(fillTypes, kind) || has(selectTypes, kind) || has(toggleTypes, kind) || has(clickTypes, kind) || kind == "XCUIElementTypeOther" && n.boolAttr("accessible", false)
		if _, skip := skipTypes[kind]; actionable && !skip && n.visible() && n.boolAttr("hittable", true) {
			if onScreen(r, clip) {
				label := n.label()
				if label != "" {
					nodeID := assign(n)
					rect := r
					base := page.Action{
						Node:  nodeID,
						Role:  roles[kind],
						Label: label,
						Rect:  &rect,
						Value: n.attr("value"),
					}
					if base.Role == "" {
						base.Role = "button"
					}
					switch {
					case has(fillTypes, kind):
						fill := base
						fill.Kind = "fill"
						actions = append(actions, fill)
						open := base
						open.Kind = "click"
						open.Label = "Open " + label
						actions = append(actions, open)
					case has(selectTypes, kind):
						current := n.attr("value")
						raw := n.attr("values", "availableValues")
						if raw != "" {
							for _, option := range strings.Split(raw, ",") {
								option = strings.TrimSpace(option)
								if option == "" || option == current {
									continue
								}
								sel := base
								sel.Kind = "select"
								sel.Label = label + " → " + option
								sel.Value = option
								sel.CurrentValue = current
								actions = append(actions, sel)
							}
						}
					case has(toggleTypes, kind):
						checked := n.attr("value")
						click := base
						click.Kind = "click"
						click.Checked = strconv.FormatBool(checked == "1" || strings.EqualFold(checked, "true"))
						actions = append(actions, click)
					case has(clickTypes, kind), kind == "XCUIElementTypeOther" && n.boolAttr("accessible", false):
						click := base
						click.Kind = "click"
						if kind == "XCUIElementTypeOther" {
							click.Role = "element"
						}
						actions = append(actions, click)
					}
				}
			}
		}
		for i := range n.Nodes {
			visit(&n.Nodes[i], clip)
		}
	}
	visit(&root, window)

	omitted := 0
	if len(actions) > page.MaxElementActions {
		omitted = len(actions) - page.MaxElementActions
		actions = actions[:page.MaxElementActions]
	}
	for i := range actions {
		actions[i].ID = "e" + strconv.Itoa(i+1)
	}
	actions = append(actions, page.Action{ID: "wait", Kind: "wait", Label: "Wait for the screen to update"})
	actions = append(actions, page.Action{ID: "home", Kind: "home", Label: "Go to Home Screen"})
	if scrollRect != nil && !hasAlert {
		actions = append(actions,
			page.Action{ID: "scroll_down", Kind: "scroll", Label: "Scroll down to reveal content below", Direction: "up", Rect: scrollRect},
			page.Action{ID: "scroll_up", Kind: "scroll", Label: "Scroll up to reveal content above", Direction: "down", Rect: scrollRect},
		)
	}
	if hasAlert {
		actions = append(actions,
			page.Action{ID: "accept_alert", Kind: "accept_alert", Label: "Accept the system alert"},
			page.Action{ID: "dismiss_alert", Kind: "dismiss_alert", Label: "Dismiss the system alert"},
		)
	}
	text := strings.Join(words, "\n")
	if len(text) > page.MaxText {
		text = text[:page.MaxText]
	}
	height := 0.0
	width := 0.0
	if window != nil {
		height, width = window.H, window.W
	}
	p := page.Page{
		URL:            "app://" + appBundle,
		Title:          appName,
		Text:           text,
		W:              width,
		H:              height,
		Scroll:         page.Scroll{Y: 0, Height: height},
		Actions:        actions,
		OmittedActions: omitted,
		BundleID:       appBundle,
		Source:         source,
	}
	return page.WithFingerprint(p), nil
}

func has(set map[string]struct{}, k string) bool {
	_, ok := set[k]
	return ok
}
