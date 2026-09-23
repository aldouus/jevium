package appium_test

import (
	"encoding/json"
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/policy"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPickerUsesVerifiedNativeReference(t *testing.T) {
	source := screen(fixtureNode{Kind: "PickerWheel", Name: "Month", Label: "Month", Value: "April", Values: "April,May", Rect: bounds(20.5, 100, 120.25, 200)})
	for _, op := range []string{"SELECT", "PICKER_NEXT", "PICKER_PREVIOUS"} {
		t.Run(op, func(t *testing.T) {
			mutations := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var value any
				switch {
				case r.URL.Path == "/session":
					value = map[string]string{"sessionId": "test"}
				case strings.HasSuffix(r.URL.Path, "/source"):
					value = source
				case strings.HasSuffix(r.URL.Path, "/elements"):
					value = []map[string]string{{"element-6066-11e4-a52e-4f735466cecf": "wheel"}}
				case strings.HasSuffix(r.URL.Path, "/attribute/name"):
					value = "Month"
				case strings.HasSuffix(r.URL.Path, "/rect"):
					value = map[string]float64{"x": 20.5, "y": 100, "width": 120.25, "height": 200}
				case strings.HasSuffix(r.URL.Path, "/value"):
					var body map[string]string
					json.NewDecoder(r.Body).Decode(&body)
					if r.URL.Path != "/session/test/element/wheel/value" || body["text"] != "May" {
						t.Errorf("request %s %v", r.URL.Path, body)
					}
					mutations++
				case strings.HasSuffix(r.URL.Path, "/execute/sync"):
					var body struct {
						Script string `json:"script"`
						Args   []struct {
							ElementID string `json:"elementId"`
							Order     string `json:"order"`
						} `json:"args"`
					}
					json.NewDecoder(r.Body).Decode(&body)
					want := "next"
					if op == "PICKER_PREVIOUS" {
						want = "previous"
					}
					if body.Script != "mobile: selectPickerWheelValue" || len(body.Args) != 1 {
						t.Errorf("body=%+v", body)
					} else if body.Args[0].ElementID != "wheel" || body.Args[0].Order != want {
						t.Errorf("args=%+v", body.Args)
					}
					mutations++
				}
				json.NewEncoder(w).Encode(map[string]any{"value": value})
			}))
			defer srv.Close()
			d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", HTTP: srv.Client()})
			if err != nil {
				t.Fatal(err)
			}
			p, err := d.Observe(false)
			if err != nil {
				t.Fatal(err)
			}
			_, targets, _ := policy.ActionSpace(p.Actions)
			if len(targets[op]) != 1 {
				t.Fatalf("targets=%v", targets)
			}
			for _, a := range targets[op] {
				if err := d.Act(a, p, nil); err != nil {
					t.Fatal(err)
				}
			}
			if mutations != 1 {
				t.Fatalf("mutations=%d", mutations)
			}
		})
	}
}
