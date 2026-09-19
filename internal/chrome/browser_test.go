package chrome_test

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/aldous/jevium/internal/chrome"
	"github.com/aldous/jevium/internal/page"
)

type rec struct {
	Method  string
	Params  map[string]any
	Session bool
}

type fakeRuntime struct {
	calls []rec
	eval  func(expr string) (any, error)
	call  func(method string, params map[string]any, session bool) (map[string]any, error)
}

func (f *fakeRuntime) Evaluate(expr string) (any, error) {
	f.calls = append(f.calls, rec{Method: "Runtime.evaluate", Params: map[string]any{"expression": expr}, Session: true})
	if f.eval != nil {
		return f.eval(expr)
	}
	return nil, nil
}

func (f *fakeRuntime) Call(method string, params map[string]any, session bool) (map[string]any, error) {
	f.calls = append(f.calls, rec{Method: method, Params: params, Session: session})
	if f.call != nil {
		return f.call(method, params, session)
	}
	return map[string]any{}, nil
}

func snapshotRaw(node float64) map[string]any {
	return map[string]any{
		"url":   "https://example.com/",
		"title": "Example Domain",
		"w":     1120.0,
		"h":     780.0,
		"text":  "Example Domain",
		"scroll": map[string]any{
			"y":      0.0,
			"height": 780.0,
		},
		"actions": []any{
			map[string]any{
				"id":    "e1",
				"kind":  "click",
				"label": "More information...",
				"role":  "link",
				"node":  node,
				"rect":  map[string]any{"x": 10.0, "y": 20.0, "w": 80.0, "h": 16.0},
			},
			map[string]any{
				"id":    "e2",
				"kind":  "fill",
				"label": "Search",
				"role":  "textbox",
				"node":  node + 1,
				"value": "",
				"rect":  map[string]any{"x": 10.0, "y": 50.0, "w": 200.0, "h": 24.0},
			},
			map[string]any{"id": "wait", "kind": "wait", "label": "Wait for the page to update"},
		},
		"marker":   []any{"origin", "https://example.com/", 0.0, 0.0, 1120.0, 780.0, "Example Domain", "Example Domain"},
		"page_key": []any{"origin", "https://example.com/", 0.0, 0.0, 1120.0, 780.0, []any{}},
		"guards": map[string]any{
			"7": []any{7.0, "link", "More information...", nil, nil, nil, nil, false, nil, nil, nil, nil, "https://iana.org/", "Example Domain"},
			"8": []any{8.0, "textbox", "Search", "", nil, nil, nil, false, nil, nil, nil, nil, nil, "Example Domain"},
		},
		"omitted_actions": 0,
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestObserveDecodesSnapshotWithoutScreenshot(t *testing.T) {
	rt := &fakeRuntime{eval: func(string) (any, error) { return snapshotRaw(7), nil }}
	b := chrome.FromRuntime(rt)
	p, err := b.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	if p.URL != "https://example.com/" || p.Title != "Example Domain" {
		t.Fatalf("page=%+v", p)
	}
	if len(p.Actions) != 3 || p.Actions[0].ID != "e1" || p.Actions[0].Kind != "click" {
		t.Fatalf("actions=%+v", p.Actions)
	}
	if fmt.Sprint(p.Actions[0].Node) != "7" {
		t.Fatalf("node=%v want observed id 7", p.Actions[0].Node)
	}
	if p.Screenshot != "" {
		t.Fatal("screenshot leaked into Observe(false)")
	}
	if p.Fingerprint == "" {
		t.Fatal("missing fingerprint")
	}
	if len(rt.calls) != 1 || rt.calls[0].Method != "Runtime.evaluate" {
		t.Fatalf("calls=%+v", rt.calls)
	}
	if strings.Contains(mustJSON(t, rt.calls[0].Params), "captureScreenshot") {
		t.Fatal("Observe(false) captured a screenshot")
	}
}

func TestObserveAttachesScreenshotOnlyWhenRequested(t *testing.T) {
	rt := &fakeRuntime{
		eval: func(string) (any, error) { return snapshotRaw(7), nil },
		call: func(method string, _ map[string]any, session bool) (map[string]any, error) {
			if method != "Page.captureScreenshot" || !session {
				t.Fatalf("unexpected call %s session=%v", method, session)
			}
			return map[string]any{"data": "jpeg-bytes"}, nil
		},
	}
	p, err := chrome.FromRuntime(rt).Observe(true)
	if err != nil {
		t.Fatal(err)
	}
	if p.Screenshot != "jpeg-bytes" {
		t.Fatalf("screenshot=%q", p.Screenshot)
	}
	if len(p.Actions) == 0 || fmt.Sprint(p.Actions[0].Node) != "7" {
		t.Fatalf("actions still need observed nodes: %+v", p.Actions)
	}
}

func TestObserveRejectsNavigatingDocument(t *testing.T) {
	rt := &fakeRuntime{eval: func(string) (any, error) { return nil, nil }}
	_, err := chrome.FromRuntime(rt).Observe(false)
	if err == nil || !strings.Contains(err.Error(), "navigating") {
		t.Fatalf("err=%v", err)
	}
}

func TestFreshUsesPageKeyAndGuardForClick(t *testing.T) {
	raw := snapshotRaw(7)
	rt := &fakeRuntime{eval: func(expr string) (any, error) {
		if strings.Contains(expr, "c.guard") {
			return []any{raw["page_key"], raw["guards"].(map[string]any)["7"]}, nil
		}
		return raw["marker"], nil
	}}
	b := chrome.FromRuntime(rt)
	p, err := decodePage(t, raw)
	if err != nil {
		t.Fatal(err)
	}
	click := p.Actions[0]
	if !b.Fresh(p, &click) {
		t.Fatal("fresh click should match observed guard")
	}
	if !strings.Contains(rt.calls[len(rt.calls)-1].Params["expression"].(string), "c.nodes.get(7)") {
		t.Fatalf("fresh must look up observed node 7, got %v", rt.calls[len(rt.calls)-1].Params)
	}
}

func TestFreshRejectsStaleClickGuard(t *testing.T) {
	raw := snapshotRaw(7)
	rt := &fakeRuntime{eval: func(expr string) (any, error) {
		if strings.Contains(expr, "c.guard") {
			return []any{raw["page_key"], []any{"stale"}}, nil
		}
		return raw["marker"], nil
	}}
	p, _ := decodePage(t, raw)
	click := p.Actions[0]
	if chrome.FromRuntime(rt).Fresh(p, &click) {
		t.Fatal("stale guard must fail")
	}
}

func TestFreshComparesMarkerWhenActionHasNoNode(t *testing.T) {
	raw := snapshotRaw(7)
	rt := &fakeRuntime{eval: func(string) (any, error) { return raw["marker"], nil }}
	p, _ := decodePage(t, raw)
	if !chrome.FromRuntime(rt).Fresh(p, nil) {
		t.Fatal("matching marker should be fresh")
	}
	rt.eval = func(string) (any, error) { return []any{"other"}, nil }
	if chrome.FromRuntime(rt).Fresh(p, nil) {
		t.Fatal("changed marker should be stale")
	}
}

func TestActRejectsStalePageWithoutMutating(t *testing.T) {
	raw := snapshotRaw(7)
	rt := &fakeRuntime{eval: func(string) (any, error) { return []any{"stale"}, nil }}
	p, _ := decodePage(t, raw)
	err := chrome.FromRuntime(rt).Act(p.Actions[0], p, nil)
	if err == nil || !strings.Contains(err.Error(), "Observe again") {
		t.Fatalf("err=%v", err)
	}
	for _, c := range rt.calls {
		if strings.HasPrefix(c.Method, "Input.") {
			t.Fatalf("mutated after stale: %+v", rt.calls)
		}
	}
}

func TestActClickDispatchesMouseOnObservedNode(t *testing.T) {
	raw := snapshotRaw(7)
	var pressed, released map[string]any
	rt := &fakeRuntime{
		eval: func(expr string) (any, error) {
			if strings.Contains(expr, "c.guard") {
				return []any{raw["page_key"], raw["guards"].(map[string]any)["7"]}, nil
			}
			if strings.Contains(expr, "nodes.get") {
				if !strings.Contains(expr, `"node":7`) && !strings.Contains(expr, `"node": 7`) {
					t.Fatalf("act must pass observed node, expr=%s", expr)
				}
				return map[string]any{"x": 50.0, "y": 28.0}, nil
			}
			return raw["marker"], nil
		},
		call: func(method string, params map[string]any, session bool) (map[string]any, error) {
			if !session {
				t.Fatalf("input must be session-scoped: %s", method)
			}
			switch params["type"] {
			case "mousePressed":
				pressed = params
			case "mouseReleased":
				released = params
			}
			return map[string]any{}, nil
		},
	}
	p, _ := decodePage(t, raw)
	if err := chrome.FromRuntime(rt).Act(p.Actions[0], p, nil); err != nil {
		t.Fatal(err)
	}
	if pressed["x"] != 50.0 || pressed["y"] != 28.0 || pressed["button"] != "left" {
		t.Fatalf("pressed=%v", pressed)
	}
	if released["x"] != 50.0 || released["y"] != 28.0 {
		t.Fatalf("released=%v", released)
	}
}

func TestActFillRequiresHelperTextAndInsertsIt(t *testing.T) {
	raw := snapshotRaw(7)
	var inserted string
	rt := &fakeRuntime{
		eval: func(expr string) (any, error) {
			if strings.Contains(expr, "isConnected") {
				return true, nil
			}
			if strings.Contains(expr, "c.guard") {
				return []any{raw["page_key"], raw["guards"].(map[string]any)["8"]}, nil
			}
			if strings.Contains(expr, "marker") {
				return raw["marker"], nil
			}
			return map[string]any{"x": 110.0, "y": 62.0}, nil
		},
		call: func(method string, params map[string]any, _ bool) (map[string]any, error) {
			if method == "Input.insertText" {
				inserted, _ = params["text"].(string)
			}
			return map[string]any{}, nil
		},
	}
	p, _ := decodePage(t, raw)
	fill := p.Actions[1]
	err := chrome.FromRuntime(rt).Act(fill, p, nil)
	if err == nil || !strings.Contains(err.Error(), "helper") {
		t.Fatalf("err=%v", err)
	}
	text := "example.com"
	if err := chrome.FromRuntime(rt).Act(fill, p, &text); err != nil {
		t.Fatal(err)
	}
	if inserted != "example.com" {
		t.Fatalf("inserted=%q", inserted)
	}
}

func TestActCoveredTargetDoesNotClick(t *testing.T) {
	raw := snapshotRaw(7)
	rt := &fakeRuntime{
		eval: func(expr string) (any, error) {
			if strings.Contains(expr, "c.guard") {
				return []any{raw["page_key"], raw["guards"].(map[string]any)["7"]}, nil
			}
			return nil, nil
		},
		call: func(string, map[string]any, bool) (map[string]any, error) {
			t.Fatal("covered target must not dispatch input")
			return nil, nil
		},
	}
	p, _ := decodePage(t, raw)
	err := chrome.FromRuntime(rt).Act(p.Actions[0], p, nil)
	if err == nil || !strings.Contains(err.Error(), "Observe again") {
		t.Fatalf("err=%v", err)
	}
}

func TestActWaitDoesNotTouchThePage(t *testing.T) {
	raw := snapshotRaw(7)
	rt := &fakeRuntime{eval: func(string) (any, error) { return raw["marker"], nil }}
	p, _ := decodePage(t, raw)
	wait := page.Action{ID: "wait", Kind: "wait", Label: "Wait for the page to update"}
	if err := chrome.FromRuntime(rt).Act(wait, p, nil); err != nil {
		t.Fatal(err)
	}
	for _, c := range rt.calls {
		if c.Method != "Runtime.evaluate" {
			t.Fatalf("wait mutated: %+v", rt.calls)
		}
	}
}

func decodePage(t *testing.T, raw map[string]any) (page.Page, error) {
	t.Helper()
	encoded, err := json.Marshal(raw)
	if err != nil {
		return page.Page{}, err
	}
	var p page.Page
	if err := json.Unmarshal(encoded, &p); err != nil {
		return page.Page{}, err
	}
	return page.WithFingerprint(p), nil
}
