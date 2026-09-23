package appium

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
	"github.com/aldous/jevium/internal/visual"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fixedOCR struct{}

func (fixedOCR) Recognize([]byte) ([]visual.Region, error) {
	return []visual.Region{{Label: "Canvas continue", X: 0.1, Y: 0.2, W: 0.4, H: 0.1}}, nil
}

func TestVisualTargetsUseObservedFrameAndRejectChangedPixels(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 750, 1600))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	frame := base64.StdEncoding.EncodeToString(buf.Bytes())
	taps := 0
	var tapped point
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value any
		switch {
		case strings.HasSuffix(r.URL.Path, "/window/rect"):
			value = windowRect{Width: 375, Height: 800}
		case strings.HasSuffix(r.URL.Path, "/source"):
			value = screen(window(375, 800))
		case strings.HasSuffix(r.URL.Path, "/screenshot"):
			value = frame
		case strings.HasSuffix(r.URL.Path, "/execute/sync"):
			var body executeRequest[point]
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			if body.Script == "mobile: tap" {
				taps++
				if len(body.Args) != 1 {
					t.Error("missing point")
				} else {
					tapped = body.Args[0]
				}
			}
		}
		json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer srv.Close()
	d, err := New(Config{URL: srv.URL, UDID: "test", SessionID: "session", HTTP: srv.Client(), Visual: fixedOCR{}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := page.FindAction(p.Actions, "visual_0")
	if !ok {
		t.Fatal("missing observed OCR target")
	}
	for _, scope := range []string{"web", "native"} {
		if _, ok := page.FindAction(policy.ScopedActions(p, scope), a.ID); ok {
			t.Fatalf("%s scope accepted OCR target with unknown content provenance", scope)
		}
	}
	if err := d.Act(a, p, nil); err != nil {
		t.Fatal(err)
	}
	if taps != 1 || tapped != (point{X: 112.5, Y: 200}) {
		t.Fatalf("taps=%d point=%+v", taps, tapped)
	}
	img.Set(0, 0, color.White)
	buf.Reset()
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	frame = base64.StdEncoding.EncodeToString(buf.Bytes())
	if err := d.Act(a, p, nil); err == nil || err.Error() != "Screen changed since this decision. Observe again." {
		t.Fatalf("changed frame error=%v", err)
	}
	if taps != 1 {
		t.Fatal("tap replayed after frame change")
	}
}
