package appium

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/visual"
	"image"
	"math"
	"net/http"
	"reflect"
)

func frameHash(encoded string) (string, []byte, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, fmt.Errorf("invalid screenshot encoding: %w", err)
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), data, nil
}

func (d *Device) visualPage(p page.Page) (page.Page, error) {
	hash, frame, err := frameHash(p.Screenshot)
	if err != nil {
		return p, err
	}
	if err := visual.Validate(frame, nil); err != nil {
		return p, err
	}
	regions, err := d.cfg.Visual.Recognize(frame)
	if err != nil {
		return p, err
	}
	if err := visual.Validate(frame, regions); err != nil {
		return p, err
	}
	if p.W <= 0 || p.H <= 0 {
		return p, fmt.Errorf("visual targets require observed window dimensions")
	}
	var window windowRect
	if err := d.call(http.MethodGet, d.path("/window/rect"), nil, &window); err != nil {
		return p, err
	}
	if window.X != 0 || window.Y != 0 || window.Width != p.W || window.Height != p.H {
		return p, fmt.Errorf("visual targets require a full-screen window with matching native dimensions")
	}
	imageConfig, _, err := image.DecodeConfig(bytes.NewReader(frame))
	if err != nil {
		return p, err
	}
	if math.Abs(float64(imageConfig.Width)/float64(imageConfig.Height)-p.W/p.H) > 0.01 {
		return p, fmt.Errorf("screenshot aspect ratio differs from observed window")
	}
	p.Guards = map[string]any{"native_fingerprint": p.Fingerprint, "visual_frame": hash}
	for i, r := range regions {
		if len(p.Actions) >= page.MaxElementActions {
			p.OmittedActions += len(regions) - i
			break
		}
		rect := page.Rect{X: r.X * p.W, Y: r.Y * p.H, W: r.W * p.W, H: r.H * p.H}
		covered := false
		for _, a := range p.Actions {
			if a.Rect != nil && rect.X+rect.W/2 >= a.Rect.X && rect.X+rect.W/2 < a.Rect.X+a.Rect.W && rect.Y+rect.H/2 >= a.Rect.Y && rect.Y+rect.H/2 < a.Rect.Y+a.Rect.H && a.Kind == "click" {
				covered = true
				break
			}
		}
		if covered {
			continue
		}
		id := fmt.Sprintf("visual_%d", i)
		p.Actions = append(p.Actions, page.Action{ID: id, Kind: "click", Scope: "visual", Node: id, Label: r.Label, Role: "visual text (clickability unknown)", Rect: &rect})
	}
	return page.WithFingerprint(p), nil
}

func (d *Device) visualFresh(p, current page.Page, a *page.Action) bool {
	if p.Guards == nil || p.Guards["native_fingerprint"] != current.Fingerprint {
		return false
	}
	var screenshot string
	if err := d.call(http.MethodGet, d.path("/screenshot"), nil, &screenshot); err != nil {
		return false
	}
	hash, _, err := frameHash(screenshot)
	if err != nil || p.Guards["visual_frame"] != hash {
		return false
	}
	if a != nil {
		observed, ok := page.FindAction(p.Actions, a.ID)
		if !ok || !reflect.DeepEqual(observed, *a) {
			return false
		}
	}
	return true
}
