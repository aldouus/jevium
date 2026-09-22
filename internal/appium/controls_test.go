package appium

import (
	"encoding/json"
	"github.com/aldous/jevium/internal/page"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestDeviceControlsFollowObservedState(t *testing.T) {
	locked := false
	orientation := "PORTRAIT"
	keyboard := true
	var mutations []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value any
		switch {
		case strings.HasSuffix(r.URL.Path, "/source"):
			source := `<AppiumAUT><XCUIElementTypeWindow width="375" height="812"/>`
			if keyboard {
				source += `<XCUIElementTypeKeyboard visible="true"/>`
			}
			value = source + `</AppiumAUT>`
		case strings.HasSuffix(r.URL.Path, "/is_locked"):
			if r.Method != "POST" {
				t.Error("lock state requires POST")
			}
			value = locked
		case strings.HasSuffix(r.URL.Path, "/orientation") && r.Method == "GET":
			value = orientation
		case strings.HasSuffix(r.URL.Path, "/orientation"):
			var body struct {
				Orientation string `json:"orientation"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			orientation = body.Orientation
			mutations = append(mutations, orientation)
		case strings.HasSuffix(r.URL.Path, "/hide_keyboard"):
			keyboard = false
			mutations = append(mutations, "hide")
		case strings.HasSuffix(r.URL.Path, "/lock"):
			locked = true
			mutations = append(mutations, "lock")
		case strings.HasSuffix(r.URL.Path, "/unlock"):
			locked = false
			mutations = append(mutations, "unlock")
		}
		json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer srv.Close()
	d, err := New(Config{URL: srv.URL, UDID: "test", SessionID: "session", HTTP: srv.Client(), DeviceControls: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"rotate_landscape", "hide_keyboard", "lock", "unlock"} {
		p, err := d.Observe(false)
		if err != nil {
			t.Fatal(err)
		}
		a, ok := page.FindAction(p.Actions, id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if err := d.Act(a, p, nil); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(mutations, []string{"LANDSCAPE", "hide", "lock", "unlock"}) {
		t.Fatalf("mutations=%v", mutations)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := page.FindAction(p.Actions, "hide_keyboard"); ok {
		t.Fatal("offered absent keyboard")
	}
	a, _ := page.FindAction(p.Actions, "rotate_portrait")
	orientation = "PORTRAIT"
	if err := d.Act(a, p, nil); err == nil {
		t.Fatal("stale orientation accepted")
	}
	if len(mutations) != 4 {
		t.Fatal("mutated stale state")
	}
}
