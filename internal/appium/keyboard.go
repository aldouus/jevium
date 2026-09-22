package appium

import (
	"github.com/aldous/jevium/internal/page"
	"net/http"
	"net/url"
	"strings"
)

func (d *Device) keyboardSnapshot(src string) (page.Page, error) {
	p, err := d.navigationSnapshot(src)
	if err != nil {
		return p, err
	}
	if !strings.Contains(src, "<XCUIElementTypeKeyboard") {
		return p, nil
	}
	var active map[string]string
	if err := d.call(http.MethodGet, d.path("/element/active"), nil, &active); err != nil {
		return p, err
	}
	id := active["element-6066-11e4-a52e-4f735466cecf"]
	if id == "" {
		return p, nil
	}
	var rect windowRect
	if err := d.call(http.MethodGet, d.path("/element/"+url.PathEscape(id)+"/rect"), nil, &rect); err != nil {
		return p, err
	}
	for _, a := range p.Actions {
		if a.Kind != "fill" || a.Rect == nil || a.Rect.X != rect.X || a.Rect.Y != rect.Y || a.Rect.W != rect.Width || a.Rect.H != rect.Height {
			continue
		}
		resolved, err := d.resolveElement(a)
		if err != nil {
			return p, err
		}
		if resolved != id {
			continue
		}
		for _, key := range []string{"backspace", "left", "right", "return", "select_all", "select_left", "select_right"} {
			edit := a
			edit.ID = "key_" + key
			edit.Kind = edit.ID
			p.Actions = append(p.Actions, edit)
		}
		break
	}
	return page.WithFingerprint(p), nil
}
