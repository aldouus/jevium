package appium

import (
	"encoding/xml"
	"github.com/aldous/jevium/internal/page"
	"net/http"
)

func (d *Device) snapshot(source string) (page.Page, error) {
	p, err := d.keyboardSnapshot(source)
	if err != nil || !d.cfg.DeviceControls {
		return p, err
	}
	var locked bool
	if err := d.call(http.MethodPost, d.path("/appium/device/is_locked"), struct{}{}, &locked); err != nil {
		return p, err
	}
	if locked {
		p.Actions = append(p.Actions, page.Action{ID: "unlock", Kind: "unlock", Label: "Request OS unlock (cannot bypass a passcode)"})
	} else {
		var orientation string
		if err := d.call(http.MethodGet, d.path("/orientation"), nil, &orientation); err != nil {
			return p, err
		}
		switch orientation {
		case "PORTRAIT":
			p.Actions = append(p.Actions, page.Action{ID: "rotate_landscape", Kind: "rotate", Value: "LANDSCAPE", Label: "Rotate device to landscape"})
		case "LANDSCAPE":
			p.Actions = append(p.Actions, page.Action{ID: "rotate_portrait", Kind: "rotate", Value: "PORTRAIT", Label: "Rotate device to portrait"})
		default:
			return p, DeviceError{Msg: "Appium returned an unknown device orientation"}
		}
		p.Actions = append(p.Actions, page.Action{ID: "lock", Kind: "lock", Label: "Lock the device"})
		var root node
		if err := xml.Unmarshal([]byte(source), &root); err != nil {
			return p, err
		}
		if visibleKeyboard(root) {
			p.Actions = append(p.Actions, page.Action{ID: "hide_keyboard", Kind: "hide_keyboard", Label: "Dismiss the on-screen keyboard"})
		}
	}
	return page.WithFingerprint(p), nil
}

func visibleKeyboard(n node) bool {
	if n.local() == "XCUIElementTypeKeyboard" && n.visible() {
		return true
	}
	for _, child := range n.Nodes {
		if visibleKeyboard(child) {
			return true
		}
	}
	return false
}

func (d *Device) control(a page.Action, p page.Page) error {
	observed, ok := page.FindAction(p.Actions, a.ID)
	if !d.cfg.DeviceControls || !ok || observed.Kind != a.Kind || observed.Value != a.Value {
		return UnsupportedError{Msg: "Device control was not observed or enabled"}
	}
	switch a.Kind {
	case "rotate":
		return d.call(http.MethodPost, d.path("/orientation"), map[string]string{"orientation": a.Value}, nil)
	case "hide_keyboard":
		return d.call(http.MethodPost, d.path("/appium/device/hide_keyboard"), struct{}{}, nil)
	case "lock":
		return d.call(http.MethodPost, d.path("/appium/device/lock"), struct{}{}, nil)
	case "unlock":
		return d.call(http.MethodPost, d.path("/appium/device/unlock"), struct{}{}, nil)
	}
	return UnsupportedError{Msg: "Unknown device control"}
}
