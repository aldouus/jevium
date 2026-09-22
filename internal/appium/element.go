package appium

import (
	"github.com/aldous/jevium/internal/page"
	"net/http"
	"net/url"
	"strings"
)

func (d *Device) resolveElement(a page.Action) (string, error) {
	node, ok := a.Node.(string)
	first, last := strings.Index(node, ":"), strings.LastIndex(node, ":")
	if !ok || first < 1 || last <= first || a.Rect == nil {
		return "", StalePageError{Msg: "Target has no native identity"}
	}
	kind, name := node[:first], node[first+1:last]
	var refs []map[string]string
	if err := d.call(http.MethodPost, d.path("/elements"), map[string]string{"using": "class name", "value": kind}, &refs); err != nil {
		return "", err
	}
	found := ""
	for _, ref := range refs {
		id := ref["element-6066-11e4-a52e-4f735466cecf"]
		if id == "" {
			continue
		}
		path := d.path("/element/" + url.PathEscape(id))
		var actual string
		if err := d.call(http.MethodGet, path+"/attribute/name", nil, &actual); err != nil {
			return "", err
		}
		if actual != name {
			continue
		}
		var r windowRect
		if err := d.call(http.MethodGet, path+"/rect", nil, &r); err != nil {
			return "", err
		}
		if r.X != a.Rect.X || r.Y != a.Rect.Y || r.Width != a.Rect.W || r.Height != a.Rect.H {
			continue
		}
		if found != "" {
			return "", StalePageError{Msg: "Native target is ambiguous"}
		}
		found = id
	}
	if found == "" {
		return "", StalePageError{Msg: "Native target is no longer present"}
	}
	return found, nil
}
