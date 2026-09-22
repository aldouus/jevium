package appium_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
)

func TestReplaceClearsObservedActiveFieldBeforeTyping(t *testing.T) {
	const source = `<AppiumAUT>
  <XCUIElementTypeTextField name="Search" value="old" focused="true"
    x="20" y="30" width="200" height="40"/>
</AppiumAUT>`
	var mutations []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value any
		switch {
		case r.URL.Path == "/session":
			value = map[string]string{"sessionId": "test"}
		case strings.HasSuffix(r.URL.Path, "/source"):
			value = source
		case strings.HasSuffix(r.URL.Path, "/element/active"):
			value = map[string]string{"element-6066-11e4-a52e-4f735466cecf": "field"}
		case strings.HasSuffix(r.URL.Path, "/rect"):
			value = map[string]int{"x": 20, "y": 30, "width": 200, "height": 40}
		case strings.HasSuffix(r.URL.Path, "/execute/sync"):
			mutations = append(mutations, "tap")
		case strings.HasSuffix(r.URL.Path, "/clear"):
			mutations = append(mutations, "clear")
		case strings.HasSuffix(r.URL.Path, "/actions"):
			mutations = append(mutations, "type")
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
	for _, op := range []string{"TYPE_TEXT", "CLEAR_TEXT", "BACKSPACE", "CURSOR_LEFT", "CURSOR_RIGHT", "RETURN"} {
		if len(targets[op]) != 1 {
			t.Fatalf("%s targets=%v", op, targets[op])
		}
	}
	var field page.Action
	for _, a := range targets["TYPE_TEXT"] {
		field = a
	}
	text := "new"
	if err := d.Act(field, p, &text); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mutations, []string{"tap", "clear", "type"}) {
		t.Fatalf("mutations=%v", mutations)
	}
}
