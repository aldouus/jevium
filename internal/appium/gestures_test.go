package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/policy"
	"strings"
	"testing"
)

func TestGroundedGesturesExecuteOnce(t *testing.T) {
	source := screen(window(400, 800,
		button("Item", bounds(20, 40, 100, 80)), button("Destination", bounds(200, 300, 100, 80)),
		fixtureNode{Kind: "Slider", Name: "Volume", Rect: bounds(20, 500, 200, 30)},
	))
	for _, op := range []string{"LONG_PRESS", "DOUBLE_TAP", "SWIPE_LEFT", "SWIPE_RIGHT", "DRAG", "PINCH_IN", "PINCH_OUT", "SET_SLIDER"} {
		t.Run(op, func(t *testing.T) {
			mutations := 0
			srv, _ := mockAppium(t, source, func(r *recorded) any {
				if strings.HasSuffix(r.Path, "/elements") {
					return []map[string]string{{"element-6066-11e4-a52e-4f735466cecf": "slider"}}
				}
				if strings.HasSuffix(r.Path, "/attribute/name") {
					return "Volume"
				}
				if strings.HasSuffix(r.Path, "/rect") {
					return map[string]int{"x": 20, "y": 500, "width": 200, "height": 30}
				}
				if strings.HasSuffix(r.Path, "/value") {
					mutations++
				}
				if r.Body["script"] != nil || r.Path == "/session/sess-1/actions" {
					mutations++
					if op == "SWIPE_LEFT" || op == "SWIPE_RIGHT" {
						args := r.Body["args"].([]any)[0].(map[string]any)
						from, to := args["fromX"].(float64), args["toX"].(float64)
						if op == "SWIPE_LEFT" && from <= to || op == "SWIPE_RIGHT" && from >= to {
							t.Errorf("wrong swipe direction: %v", args)
						}
					}
					if op == "PINCH_IN" || op == "PINCH_OUT" {
						sources := r.Body["actions"].([]any)
						if len(sources) != 2 {
							t.Errorf("pinch fingers=%d", len(sources))
						}
					}
				}
				return nil
			})
			d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", HTTP: srv.Client()})
			if err != nil {
				t.Fatal(err)
			}
			p, err := d.Observe(false)
			if err != nil {
				t.Fatal(err)
			}
			_, targets, _ := policy.ActionSpace(p.Actions)
			if len(targets[op]) == 0 {
				t.Fatalf("missing %s", op)
			}
			for _, a := range targets[op] {
				if err := d.Act(a, p, nil); err != nil {
					t.Fatal(err)
				}
				break
			}
			if mutations != 1 {
				t.Fatalf("mutations=%d", mutations)
			}
		})
	}
}

func TestClippedTargetsDoNotOfferGestures(t *testing.T) {
	source := screen(window(100, 200, button("Clipped", bounds(-20, 20, 100, 50))))
	p, err := appium.SnapshotFromSource(source, "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, targets, _ := policy.ActionSpace(p.Actions)
	if len(targets["CLICK"]) != 1 {
		t.Fatal("fixture must offer center tap")
	}
	for _, op := range []string{"LONG_PRESS", "DOUBLE_TAP", "SWIPE_LEFT", "SWIPE_RIGHT", "PINCH_IN", "PINCH_OUT"} {
		if len(targets[op]) != 0 {
			t.Fatalf("clipped target offered %s", op)
		}
	}
}
