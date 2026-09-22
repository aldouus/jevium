package appium

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aldous/jevium/internal/env"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/stale"
)

type StalePageError struct{ Msg string }

func (e StalePageError) Error() string { return e.Msg }

func (e StalePageError) Is(target error) bool {
	if _, ok := target.(stale.Error); ok {
		return true
	}
	_, ok := target.(StalePageError)
	return ok
}

type DeviceError struct{ Msg string }

func (e DeviceError) Error() string { return e.Msg }

type UnsupportedError struct{ Msg string }

func (e UnsupportedError) Error() string { return e.Msg }

type Config struct {
	URL                string
	UDID               string
	BundleID           string
	SessionID          string
	WDALocalPort       int
	MJPEGServerPort    int
	XcodeOrgID         string
	XcodeSigningID     string
	UpdatedWDABundleID string
	DerivedDataPath    string
	NoReset            bool
	HTTP               *http.Client
}

type Device struct {
	cfg          Config
	sessionID    string
	ownedSession bool
	http         *http.Client
}

func New(cfg Config) (*Device, error) {
	if cfg.URL == "" {
		cfg.URL = env.Get("APPIUM_URL", "http://127.0.0.1:4723")
	}
	cfg.URL = strings.TrimRight(cfg.URL, "/")
	if cfg.UDID == "" {
		cfg.UDID = env.Get("APPIUM_UDID", "")
	}
	if cfg.UDID == "" {
		return nil, fmt.Errorf("APPIUM_UDID is required for appium mode. Export it in the shell, set it in .env, or pass --udid")
	}
	if cfg.WDALocalPort == 0 {
		cfg.WDALocalPort = env.Atoi("APPIUM_WDA_LOCAL_PORT", 8101)
	}
	if cfg.MJPEGServerPort == 0 {
		cfg.MJPEGServerPort = env.Atoi("APPIUM_MJPEG_SERVER_PORT", 9101)
	}
	if cfg.XcodeOrgID == "" {
		cfg.XcodeOrgID = env.Get("APPIUM_XCODE_ORG_ID", "")
	}
	if cfg.XcodeSigningID == "" {
		cfg.XcodeSigningID = env.Get("APPIUM_XCODE_SIGNING_ID", "Apple Development")
	}
	if cfg.UpdatedWDABundleID == "" {
		cfg.UpdatedWDABundleID = env.Get("APPIUM_UPDATED_WDA_BUNDLE_ID", "")
	}
	if cfg.DerivedDataPath == "" {
		cfg.DerivedDataPath = env.Get("APPIUM_DERIVED_DATA_PATH", "")
	}
	client := cfg.HTTP
	if client == nil {
		client = &http.Client{
			Timeout: 180 * time.Second,
			Transport: &http.Transport{
				DisableKeepAlives: true,
			},
		}
	}
	d := &Device{cfg: cfg, http: client, sessionID: cfg.SessionID}
	if cfg.SessionID != "" {
		var rect windowRect
		if err := d.call(http.MethodGet, d.path("/window/rect"), nil, &rect); err != nil {
			return nil, err
		}
		if err := d.enableHitTesting(); err != nil {
			return nil, err
		}
		return d, nil
	}
	if err := d.createSession(); err != nil {
		return nil, err
	}
	if err := d.enableHitTesting(); err != nil {
		_ = d.Close()
		return nil, err
	}
	return d, nil
}

func (d *Device) enableHitTesting() error {
	return d.call(http.MethodPost, d.path("/appium/settings"), map[string]any{
		"settings": map[string]bool{"includeHittableInPageSource": true},
	}, nil)
}

func (d *Device) path(suffix string) string { return "/session/" + d.sessionID + suffix }

func (d *Device) call(method, path string, body any, dest any) error {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, d.cfg.URL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := d.http.Do(req)
	if err != nil {
		return DeviceError{Msg: err.Error()}
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return DeviceError{Msg: err.Error()}
	}
	return decodeWire(resp.StatusCode, method, path, data, dest)
}

func (d *Device) createSession() error {
	caps := sessionCaps{
		PlatformName:       "iOS",
		AutomationName:     "XCUITest",
		UDID:               d.cfg.UDID,
		XcodeOrgID:         d.cfg.XcodeOrgID,
		XcodeSigningID:     d.cfg.XcodeSigningID,
		UpdatedWDABundleID: d.cfg.UpdatedWDABundleID,
		WDALocalPort:       d.cfg.WDALocalPort,
		MJPEGServerPort:    d.cfg.MJPEGServerPort,
		WDALaunchTimeout:   180000,
		NewCommandTimeout:  3600,
		NoReset:            true,
	}
	if d.cfg.BundleID != "" && d.cfg.BundleID != "com.apple.springboard" {
		caps.BundleID = d.cfg.BundleID
	}
	if d.cfg.DerivedDataPath != "" {
		caps.DerivedDataPath = d.cfg.DerivedDataPath
	}
	var req newSessionRequest
	req.Capabilities.AlwaysMatch = caps
	var sess sessionValue
	if err := d.call(http.MethodPost, "/session", req, &sess); err != nil {
		return err
	}
	if sess.SessionID == "" {
		return DeviceError{Msg: "Appium session response did not include sessionId"}
	}
	d.sessionID = sess.SessionID
	d.ownedSession = true
	if d.cfg.BundleID == "com.apple.springboard" {
		_ = d.activate("com.apple.springboard")
	}
	return nil
}

func (d *Device) execute[T any](script string, args []T) error {
	return d.call(http.MethodPost, d.path("/execute/sync"), executeRequest[T]{Script: script, Args: args}, nil)
}

func (d *Device) activate(bundleID string) error {
	return d.execute("mobile: activateApp", []bundleArg{{BundleID: bundleID}})
}

func (d *Device) tap(x, y float64) error {
	return d.execute("mobile: tap", []point{{X: x, Y: y}})
}

func (d *Device) source() (string, error) {
	var s string
	if err := d.call(http.MethodGet, d.path("/source"), nil, &s); err != nil {
		return "", err
	}
	if s == "" {
		return "", StalePageError{Msg: "Appium source was not XML"}
	}
	return s, nil
}

func (d *Device) Observe(screenshot bool) (page.Page, error) {
	src, err := d.source()
	if err != nil {
		return page.Page{}, err
	}
	p, err := d.snapshot(src)
	if err != nil {
		return page.Page{}, err
	}
	if screenshot {
		var s string
		if err := d.call(http.MethodGet, d.path("/screenshot"), nil, &s); err != nil {
			return page.Page{}, err
		}
		if s == "" {
			return page.Page{}, StalePageError{Msg: "Appium screenshot was not a string"}
		}
		p.Screenshot = s
	}
	return p, nil
}

func (d *Device) Fresh(p page.Page, action *page.Action) bool {
	src, err := d.source()
	if err != nil {
		return false
	}
	current, err := d.snapshot(src)
	if err != nil {
		return false
	}
	if current.Fingerprint != p.Fingerprint {
		return false
	}
	if action == nil {
		return true
	}
	cur, ok := matchingAction(current, *action)
	if !ok {
		return false
	}
	return sameRect(action.Rect, cur.Rect)
}

func matchingAction(p page.Page, action page.Action) (page.Action, bool) {
	for _, a := range p.Actions {
		if a.Kind == action.Kind && a.Label == action.Label && sameRect(action.Rect, a.Rect) {
			return a, true
		}
	}
	return page.Action{}, false
}

func sameRect(a, b *page.Rect) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.X == b.X && a.Y == b.Y && a.W == b.W && a.H == b.H
}

func (d *Device) Act(action page.Action, p page.Page, text *string) error {
	if !d.Fresh(p, nil) {
		return StalePageError{Msg: "Screen changed since this decision. Observe again."}
	}
	switch action.Kind {
	case "key_select_all", "key_select_left", "key_select_right":
		id, err := d.resolveElement(action)
		if err != nil {
			return err
		}
		if err := d.requireActive(id); err != nil {
			return err
		}
		key, flags := "a", 1<<4
		if action.Kind == "key_select_left" {
			key, flags = "XCUIKeyboardKeyLeftArrow", 1<<1
		}
		if action.Kind == "key_select_right" {
			key, flags = "XCUIKeyboardKeyRightArrow", 1<<1
		}
		return d.execute("mobile: keys", []map[string]any{{"elementId": id, "keys": []map[string]any{{"key": key, "modifierFlags": flags}}}})
	case "key_backspace", "key_left", "key_right", "key_return":
		keys := map[string]string{"key_backspace": "XCUIKeyboardKeyDelete", "key_left": "XCUIKeyboardKeyLeftArrow", "key_right": "XCUIKeyboardKeyRightArrow", "key_return": "XCUIKeyboardKeyReturn"}
		id, err := d.resolveElement(action)
		if err != nil {
			return err
		}
		if err := d.requireActive(id); err != nil {
			return err
		}
		return d.execute("mobile: keys", []map[string]any{{"elementId": id, "keys": []string{keys[action.Kind]}}})
	case "clear_text":
		id, err := d.focusField(action)
		if err != nil {
			return err
		}
		return d.call(http.MethodPost, d.path("/element/"+id+"/clear"), struct{}{}, nil)
	case "wait":
		time.Sleep(100 * time.Millisecond)
		return nil
	case "home":
		return d.activate("com.apple.springboard")
	case "accept_alert":
		return d.call(http.MethodPost, d.path("/alert/accept"), struct{}{}, nil)
	case "dismiss_alert":
		return d.call(http.MethodPost, d.path("/alert/dismiss"), struct{}{}, nil)
	case "scroll":
		r := action.Rect
		if r == nil || r.W <= 0 || r.H <= 0 {
			return StalePageError{Msg: "Scroll container is no longer visible"}
		}
		from, to := r.Y+r.H*0.75, r.Y+r.H*0.25
		switch action.Direction {
		case "up":
		case "down":
			from, to = to, from
		default:
			return UnsupportedError{Msg: "Unsupported scroll direction"}
		}
		return d.execute("mobile: dragFromToForDuration", []dragArg{{FromX: r.X + r.W/2, ToX: r.X + r.W/2, FromY: from, ToY: to, Duration: 0.1}})
	case "fill":
		if text == nil || strings.TrimSpace(*text) == "" {
			return fmt.Errorf("TYPE_TEXT needs text from the helper; the executor does not guess")
		}
		id, err := d.focusField(action)
		if err != nil {
			return err
		}
		if err := d.call(http.MethodPost, d.path("/element/"+id+"/clear"), struct{}{}, nil); err != nil {
			return err
		}
		if err := d.requireActive(id); err != nil {
			return err
		}
		return d.typeText(*text)
	case "select":
		id, err := d.resolveElement(action)
		if err != nil {
			return err
		}
		return d.call(http.MethodPost, d.path("/element/"+url.PathEscape(id)+"/value"), map[string]string{"text": action.Value}, nil)
	case "picker_next", "picker_previous":
		id, err := d.resolveElement(action)
		if err != nil {
			return err
		}
		order := "next"
		if action.Kind == "picker_previous" {
			order = "previous"
		}
		return d.execute("mobile: selectPickerWheelValue", []map[string]any{{"elementId": id, "order": order, "offset": 0.15}})
	case "click":
		return d.click(action)
	default:
		return UnsupportedError{Msg: fmt.Sprintf("unsupported action kind %s", action.Kind)}
	}
}

func (d *Device) click(action page.Action) error {
	x, y, err := tapPoint(action)
	if err != nil {
		return err
	}
	return d.tap(x, y)
}

func (d *Device) focusField(action page.Action) (string, error) {
	id, err := d.resolveElement(action)
	if err != nil {
		return "", err
	}
	if err := d.click(action); err != nil {
		return "", err
	}
	if err := d.requireActive(id); err != nil {
		return "", err
	}
	return id, nil
}

func (d *Device) requireActive(id string) error {
	var element map[string]string
	if err := d.call(http.MethodGet, d.path("/element/active"), nil, &element); err != nil {
		return DeviceError{Msg: "Could not verify focused field; not retrying"}
	}
	if element["element-6066-11e4-a52e-4f735466cecf"] != id {
		return DeviceError{Msg: "Focused field does not match resolved target; not retrying"}
	}
	return nil
}

func (d *Device) typeText(text string) error {
	return d.call(http.MethodPost, d.path("/actions"), w3cActionsRequest{
		Actions: []w3cSource{{
			Type:    "key",
			ID:      "keyboard",
			Actions: keyActions(text),
		}},
	}, nil)
}

func keyActions(text string) []w3cTick {
	out := make([]w3cTick, 0, len(text)*2)
	for _, r := range text {
		s := string(r)
		out = append(out,
			w3cTick{Type: "keyDown", Value: s},
			w3cTick{Type: "keyUp", Value: s},
		)
	}
	return out
}

func tapPoint(action page.Action) (float64, float64, error) {
	if action.Rect == nil || action.Rect.W <= 0 || action.Rect.H <= 0 {
		return 0, 0, StalePageError{Msg: "Target changed or is covered. Observe again."}
	}
	return action.Rect.X + action.Rect.W/2, action.Rect.Y + action.Rect.H/2, nil
}

func (d *Device) Close() error {
	if d.ownedSession && d.sessionID != "" {
		_ = d.call(http.MethodDelete, d.path(""), nil, nil)
		d.sessionID = ""
		d.ownedSession = false
	}
	return nil
}

func (d *Device) SessionID() string { return d.sessionID }
func (d *Device) Owned() bool       { return d.ownedSession }
