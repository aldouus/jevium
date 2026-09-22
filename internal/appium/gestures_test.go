package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/policy"
	"testing"
)

func TestGroundedGesturesExecuteOnce(t *testing.T) {
	const source = `<AppiumAUT>
  <XCUIElementTypeWindow width="400" height="800">
    <XCUIElementTypeButton name="Item" x="20" y="40" width="100" height="80"/>
    <XCUIElementTypeButton name="Destination" x="200" y="300" width="100" height="80"/>
    <XCUIElementTypeSlider name="Volume" x="20" y="500" width="200" height="30"/>
  </XCUIElementTypeWindow>
</AppiumAUT>`
	for _, op := range []string{"LONG_PRESS", "DOUBLE_TAP", "SWIPE_LEFT", "SWIPE_RIGHT", "DRAG", "PINCH_IN", "PINCH_OUT", "SET_SLIDER"} {
		t.Run(op, func(t *testing.T) {
			mutations := 0
			srv, _ := mockAppium(t, source, func(r *recorded) any {
				if r.Body["script"] != nil || r.Path == "/session/sess-1/actions" {
					mutations++
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
