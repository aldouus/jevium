package appium_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/page"
)

const oneIcon = `<?xml version="1.0" encoding="UTF-8"?>
<AppiumAUT>
  <XCUIElementTypeApplication name="SpringBoard" bundleId="com.apple.springboard" visible="true">
    <XCUIElementTypeWindow visible="true" enabled="true">
      <XCUIElementTypeIcon name="Safari" label="Safari" enabled="true" visible="true"
        accessible="true" x="24" y="80" width="60" height="60"/>
    </XCUIElementTypeWindow>
  </XCUIElementTypeApplication>
</AppiumAUT>`

type recorded struct {
	Method string
	Path   string
	Body   map[string]any
}

func mockAppium(t *testing.T, source string, onPost func(*recorded) any) (*httptest.Server, *[]recorded) {
	t.Helper()
	var calls []recorded
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		item := recorded{Method: r.Method, Path: r.URL.Path}
		if r.Body != nil {
			raw, _ := io.ReadAll(r.Body)
			if len(raw) > 0 {
				_ = json.Unmarshal(raw, &item.Body)
			}
		}
		calls = append(calls, item)
		var value any
		switch {
		case strings.HasSuffix(r.URL.Path, "/elements"):
			value = []map[string]string{{"element-6066-11e4-a52e-4f735466cecf": "field"}}
		case strings.HasSuffix(r.URL.Path, "/element/active"):
			value = map[string]string{"element-6066-11e4-a52e-4f735466cecf": "field"}
		case strings.HasSuffix(r.URL.Path, "/attribute/name"):
			value = "Address"
		case strings.Contains(r.URL.Path, "/element/") && strings.HasSuffix(r.URL.Path, "/rect"):
			value = map[string]int{"x": 48, "y": 54, "width": 240, "height": 32}
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			value = map[string]any{"sessionId": "sess-1"}
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/source"):
			value = source
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/rect"):
			value = map[string]any{"x": 30, "y": 90, "width": 50, "height": 50}
		case r.Method == http.MethodPost && onPost != nil:
			value = onPost(&item)
		default:
			value = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	t.Cleanup(srv.Close)
	return srv, &calls
}

func TestCreateSessionPostsCapabilities(t *testing.T) {
	srv, calls := mockAppium(t, oneIcon, nil)
	d, err := appium.New(appium.Config{
		URL: srv.URL, UDID: "00008110-000610D822C2401E", BundleID: "com.apple.springboard",
		WDALocalPort: 8101, MJPEGServerPort: 9101, XcodeOrgID: "Y749H8Y94K",
		UpdatedWDABundleID: "com.aldouswaites.WebDriverAgentRunner",
		DerivedDataPath:    "/tmp/Appium-iPhone13mini",
		HTTP:               srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if d.SessionID() != "sess-1" || !d.Owned() {
		t.Fatalf("session=%s owned=%v", d.SessionID(), d.Owned())
	}
	if (*calls)[0].Method != "POST" || (*calls)[0].Path != "/session" {
		t.Fatalf("first call %+v", (*calls)[0])
	}
	caps := (*calls)[0].Body["capabilities"].(map[string]any)["alwaysMatch"].(map[string]any)
	if caps["appium:udid"] != "00008110-000610D822C2401E" {
		t.Fatalf("udid=%v", caps["appium:udid"])
	}
}

func TestObserveIsOneSourceRead(t *testing.T) {
	srv, calls := mockAppium(t, oneIcon, nil)
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.springboard", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	n := len(*calls)
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls)-n != 1 {
		t.Fatalf("observe calls=%d", len(*calls)-n)
	}
	if (*calls)[n].Path != "/session/sess-1/source" {
		t.Fatalf("path=%s", (*calls)[n].Path)
	}
	found := false
	for _, a := range p.Actions {
		if a.Label == "Safari" {
			found = true
		}
	}
	if !found {
		t.Fatal("Safari missing")
	}
	if p.Screenshot != "" {
		t.Fatal("screenshot leaked")
	}
}

func TestObservedScrollExecutesOneSwipe(t *testing.T) {
	source := `<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeScrollView visible="true" x="240" y="100" width="100" height="400"/></XCUIElementTypeWindow></AppiumAUT>`
	srv, calls := mockAppium(t, source, nil)
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	a, ok := page.FindAction(p.Actions, "scroll_down")
	if !ok {
		t.Fatal("scroll_down missing")
	}
	if err := d.Act(a, p, nil); err != nil {
		t.Fatal(err)
	}
	swipes := 0
	settings := 0
	for _, call := range *calls {
		if strings.HasSuffix(call.Path, "/appium/settings") {
			settings++
			if call.Body["settings"].(map[string]any)["includeHittableInPageSource"] != true {
				t.Fatal("hit testing not enabled")
			}
		}
		if call.Body["script"] == "mobile: dragFromToForDuration" {
			swipes++
			args := call.Body["args"].([]any)
			if len(args) != 1 {
				t.Fatalf("swipe args=%v", args)
			}
			gesture := args[0].(map[string]any)
			if gesture["fromX"] != float64(290) || gesture["toX"] != float64(290) || gesture["fromY"] != float64(400) || gesture["toY"] != float64(200) {
				t.Fatalf("swipe args=%v", args)
			}
		}
	}
	if swipes != 1 || settings != 1 {
		t.Fatalf("swipes=%d settings=%d", swipes, settings)
	}
}

func TestActRejectsStalePage(t *testing.T) {
	current := oneIcon
	armed := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value any
		switch {
		case r.URL.Path == "/session":
			value = map[string]any{"sessionId": "sess-1"}
		case strings.HasSuffix(r.URL.Path, "/source"):
			value = current
		case strings.Contains(r.URL.Path, "/execute/") && armed:
			t.Fatal("mutation on stale page")
		default:
			value = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer srv.Close()
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.springboard", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	current = strings.ReplaceAll(oneIcon, "Safari", "Photos")
	armed = true
	safari := p.Actions[0]
	err = d.Act(safari, p, nil)
	if err == nil {
		t.Fatal("expected stale")
	}
}

func TestClickTapsCurrentGeometry(t *testing.T) {
	var tapped []any
	srv, _ := mockAppium(t, oneIcon, func(r *recorded) any {
		if r.Body["script"] == "mobile: tap" {
			tapped = r.Body["args"].([]any)
		}
		return nil
	})
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.springboard", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	var safari = p.Actions[0]
	if err := d.Act(safari, p, nil); err != nil {
		t.Fatal(err)
	}
	if len(tapped) != 1 {
		t.Fatalf("tapped=%v", tapped)
	}
	pt := tapped[0].(map[string]any)
	if pt["x"] != 54.0 || pt["y"] != 110.0 {
		t.Fatalf("point=%v rect=%+v", pt, safari.Rect)
	}
}

func TestFillTapsThenTypesWithW3CActions(t *testing.T) {
	const fieldXML = `<?xml version="1.0"?><AppiumAUT>
      <XCUIElementTypeApplication name="Safari" bundleId="com.apple.mobilesafari" visible="true">
        <XCUIElementTypeTextField name="Address" label="Address" value="" enabled="true"
          visible="true" accessible="true" x="48" y="54" width="240" height="32"/>
      </XCUIElementTypeApplication></AppiumAUT>`
	var taps []map[string]any
	var typed []any
	srv, calls := mockAppium(t, fieldXML, func(r *recorded) any {
		if r.Body["script"] == "mobile: tap" {
			if args, ok := r.Body["args"].([]any); ok && len(args) == 1 {
				taps = append(taps, args[0].(map[string]any))
			}
		}
		if r.Path == "/session/sess-1/actions" {
			typed = r.Body["actions"].([]any)
		}
		if strings.Contains(r.Path, "/w3c/actions") || strings.HasSuffix(r.Path, "/keys") {
			t.Fatalf("legacy type path %s", r.Path)
		}
		return nil
	})
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.mobilesafari", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	var field page.Action
	for _, a := range p.Actions {
		if a.Kind == "fill" && a.Label == "Address" {
			field = a
			break
		}
	}
	if field.Kind == "" {
		t.Fatal("missing fill action")
	}
	text := "https://example.com"
	if err := d.Act(field, p, &text); err != nil {
		t.Fatal(err)
	}
	if len(taps) != 1 {
		t.Fatalf("taps=%v calls=%v", taps, paths(*calls))
	}
	if taps[0]["x"] != 168.0 || taps[0]["y"] != 70.0 {
		t.Fatalf("tap=%v rect=%+v", taps[0], field.Rect)
	}
	if len(typed) != 1 {
		t.Fatalf("actions=%v", typed)
	}
	kb := typed[0].(map[string]any)
	if kb["type"] != "key" || kb["id"] != "keyboard" {
		t.Fatalf("keyboard=%v", kb)
	}
	seq := kb["actions"].([]any)
	if len(seq) < 2 {
		t.Fatalf("seq=%v", seq)
	}
	first := seq[0].(map[string]any)
	if first["type"] != "keyDown" || first["value"] != "h" {
		t.Fatalf("first=%v", first)
	}
	got := ""
	for _, step := range seq {
		m := step.(map[string]any)
		if m["type"] == "keyDown" {
			ch := m["value"].(string)
			if ch == "\uE007" {
				t.Fatal("fill must not send Enter")
			}
			got += ch
		}
	}
	if got != text {
		t.Fatalf("typed %q want %q", got, text)
	}
	sawTap, sawSourceAfterTap, sawType := false, false, false
	for _, c := range *calls {
		if c.Path == "/session/sess-1/execute/sync" && c.Body["script"] == "mobile: tap" {
			sawTap = true
			sawSourceAfterTap = false
		}
		if sawTap && !sawType && c.Method == "GET" && c.Path == "/session/sess-1/element/active" {
			sawSourceAfterTap = true
		}
		if c.Path == "/session/sess-1/actions" {
			if !sawSourceAfterTap {
				t.Fatal("typed without verifying active element after tap")
			}
			sawType = true
		}
	}
	if !sawType {
		t.Fatal("missing W3C type")
	}
}

func paths(calls []recorded) []string {
	out := make([]string, len(calls))
	for i, c := range calls {
		out[i] = c.Method + " " + c.Path
	}
	return out
}

func TestHTTPErrorIsNotSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": map[string]any{"error": "session not created", "message": "locked"}})
	}))
	defer srv.Close()
	_, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.springboard", HTTP: srv.Client()})
	if err == nil {
		t.Fatal("expected error")
	}
	var de appium.DeviceError
	if !errors.As(err, &de) {
		t.Fatalf("want DeviceError got %T %v", err, err)
	}
	if !strings.Contains(de.Error(), "locked") {
		t.Fatalf("err=%v", err)
	}
}

func TestValueErrorMustBeAString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": map[string]any{"error": map[string]any{"code": 1}}})
	}))
	defer srv.Close()
	_, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.springboard", HTTP: srv.Client()})
	if err == nil {
		t.Fatal("expected error")
	}
	var de appium.DeviceError
	if !errors.As(err, &de) {
		t.Fatalf("want DeviceError got %T %v", err, err)
	}
}

func TestSessionResponseWithoutSessionIDFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": map[string]any{"capabilities": map[string]any{}}})
	}))
	defer srv.Close()
	_, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.springboard", HTTP: srv.Client()})
	if err == nil {
		t.Fatal("expected error")
	}
	var de appium.DeviceError
	if !errors.As(err, &de) {
		t.Fatalf("want DeviceError got %T %v", err, err)
	}
}

func TestUnexpectedJSONFailsClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "not json")
	}))
	defer srv.Close()
	_, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.springboard", HTTP: srv.Client()})
	if err == nil {
		t.Fatal("expected error")
	}
	var de appium.DeviceError
	if !errors.As(err, &de) {
		t.Fatalf("want DeviceError got %T %v", err, err)
	}
}

func TestFillTypesAfterKeyboardShiftsSameField(t *testing.T) {
	const fieldXML = `<?xml version="1.0"?><AppiumAUT>
      <XCUIElementTypeApplication name="Safari" bundleId="com.apple.mobilesafari" visible="true">
        <XCUIElementTypeTextField name="Address" label="Address" value="" enabled="true"
          visible="true" accessible="true" x="48" y="54" width="240" height="32"/>
      </XCUIElementTypeApplication></AppiumAUT>`
	const movedXML = `<?xml version="1.0"?><AppiumAUT>
      <XCUIElementTypeApplication name="Safari" bundleId="com.apple.mobilesafari" visible="true">
        <XCUIElementTypeTextField name="Address" label="Address" value="" enabled="true"
          visible="true" accessible="true" x="48" y="200" width="240" height="32"/>
      </XCUIElementTypeApplication></AppiumAUT>`
	source := fieldXML
	var typed bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value any
		switch {
		case strings.HasSuffix(r.URL.Path, "/elements"):
			value = []map[string]string{{"element-6066-11e4-a52e-4f735466cecf": "field"}}
		case strings.HasSuffix(r.URL.Path, "/element/active"):
			value = map[string]string{"element-6066-11e4-a52e-4f735466cecf": "field"}
		case strings.HasSuffix(r.URL.Path, "/attribute/name"):
			value = "Address"
		case strings.Contains(r.URL.Path, "/element/") && strings.HasSuffix(r.URL.Path, "/rect"):
			value = map[string]int{"x": 48, "y": 54, "width": 240, "height": 32}
		case r.Method == http.MethodPost && r.URL.Path == "/session":
			value = map[string]any{"sessionId": "sess-1"}
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/source"):
			value = source
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/execute/sync"):
			source = movedXML
			value = nil
		case r.Method == http.MethodPost && r.URL.Path == "/session/sess-1/actions":
			typed = true
			value = nil
		default:
			value = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer srv.Close()
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.mobilesafari", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	var field page.Action
	for _, a := range p.Actions {
		if a.Kind == "fill" && a.Label == "Address" {
			field = a
			break
		}
	}
	if field.Kind == "" {
		t.Fatal("missing fill action")
	}
	text := "https://example.com"
	if err := d.Act(field, p, &text); err != nil {
		t.Fatal(err)
	}
	if !typed {
		t.Fatal("did not type after keyboard shifted the same field")
	}
}

func TestSelectIsUnsupported(t *testing.T) {
	const pickerXML = `<?xml version="1.0"?><AppiumAUT>
      <XCUIElementTypeApplication name="Safari" bundleId="com.apple.mobilesafari" visible="true">
        <XCUIElementTypePickerWheel name="Month" label="Month" value="September" enabled="true"
          visible="true" accessible="true" x="20" y="500" width="160" height="120"
          values="January,February,March"/>
      </XCUIElementTypeApplication></AppiumAUT>`
	srv, calls := mockAppium(t, pickerXML, func(r *recorded) any {
		if strings.Contains(r.Path, "/element") {
			t.Fatalf("element-id path %s", r.Path)
		}
		return nil
	})
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "x", BundleID: "com.apple.mobilesafari", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	var sel page.Action
	for _, a := range p.Actions {
		if a.Kind == "select" {
			sel = a
			break
		}
	}
	if sel.Kind == "" {
		t.Fatal("missing select action")
	}
	err = d.Act(sel, p, nil)
	if err == nil {
		t.Fatal("expected unsupported")
	}
	var ue appium.UnsupportedError
	if !errors.As(err, &ue) {
		t.Fatalf("want UnsupportedError got %T %v", err, err)
	}
	for _, c := range *calls {
		if strings.Contains(c.Path, "/element") {
			t.Fatalf("select used element-id path %s", c.Path)
		}
	}
}
