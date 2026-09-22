package chrome

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/aldous/jevium/internal/env"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/stale"
	"github.com/gorilla/websocket"
)

//go:embed snapshot.js
var snapshotFS embed.FS

func snapshotJS() string {
	b, _ := snapshotFS.ReadFile("snapshot.js")
	return string(b)
}

type Runtime interface {
	Evaluate(expression string) (any, error)
	Call(method string, params map[string]any, session bool) (map[string]any, error)
}

type wsRuntime struct{ b *Browser }

func (w wsRuntime) Evaluate(expression string) (any, error) {
	return w.b.evaluateWS(expression)
}

func (w wsRuntime) Call(method string, params map[string]any, session bool) (map[string]any, error) {
	if session {
		return w.b.roundTrip(method, params, w.b.session)
	}
	return w.b.roundTrip(method, params, "")
}

type Browser struct {
	ws      *websocket.Conn
	rt      Runtime
	mu      sync.Mutex
	nextID  int
	session string
	target  string
}

func FromRuntime(rt Runtime) *Browser {
	return &Browser{rt: rt}
}

func New(startURL string) (*Browser, error) {
	port := env.Get("CHROME_DEBUG_PORT", "9222")
	wsURL, err := debuggerWebSocket("http://127.0.0.1:" + port)
	if err != nil {
		return nil, err
	}
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("chrome debugger: %w (start Chrome with --remote-debugging-port=%s)", err, port)
	}
	b := &Browser{ws: conn, nextID: 1}
	b.rt = wsRuntime{b: b}
	created, err := b.call("Target.createTarget", map[string]any{"url": "about:blank", "background": true})
	if err != nil {
		conn.Close()
		return nil, err
	}
	b.target = str(created["targetId"])
	attached, err := b.call("Target.attachToTarget", map[string]any{"targetId": b.target, "flatten": true})
	if err != nil {
		conn.Close()
		return nil, err
	}
	b.session = str(attached["sessionId"])
	_, _ = b.sessionCall("Emulation.setDeviceMetricsOverride", map[string]any{
		"width": 1120, "height": 780, "deviceScaleFactor": 1, "mobile": false,
	})
	_, _ = b.sessionCall("Emulation.setFocusEmulationEnabled", map[string]any{"enabled": true})
	if _, err := b.sessionCall("Page.enable", map[string]any{}); err != nil {
		conn.Close()
		return nil, err
	}
	nav, err := b.sessionCall("Page.navigate", map[string]any{"url": startURL})
	if err != nil {
		conn.Close()
		return nil, err
	}
	if errText, _ := nav["errorText"].(string); errText != "" {
		conn.Close()
		return nil, fmt.Errorf("chrome navigation to %s failed: %s", startURL, errText)
	}
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		href, herr := b.evaluate("location.href")
		ready, rerr := b.evaluate("document.readyState")
		if herr != nil || rerr != nil {
			time.Sleep(20 * time.Millisecond)
			continue
		}
		h, r := str(href), str(ready)
		if r == "complete" && h != "" && h != "about:blank" && !strings.HasPrefix(h, "chrome-error://") {
			return b, nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	conn.Close()
	return nil, fmt.Errorf("chrome navigation to %s did not complete", startURL)
}

func debuggerWebSocket(origin string) (string, error) {
	resp, err := http.Get(origin + "/json/version")
	if err != nil {
		return "", fmt.Errorf("chrome debugger not reachable at %s: %w", origin, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	var v map[string]any
	if err := json.Unmarshal(data, &v); err != nil {
		return "", err
	}
	if ws, ok := v["webSocketDebuggerUrl"].(string); ok && ws != "" {
		return ws, nil
	}
	return "", fmt.Errorf("chrome debugger websocket missing")
}

func (b *Browser) call(method string, params map[string]any) (map[string]any, error) {
	return b.roundTrip(method, params, "")
}

func (b *Browser) sessionCall(method string, params map[string]any) (map[string]any, error) {
	return b.roundTrip(method, params, b.session)
}

func (b *Browser) roundTrip(method string, params map[string]any, session string) (map[string]any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	msg := map[string]any{"id": b.nextID, "method": method, "params": params}
	if session != "" {
		msg["sessionId"] = session
	}
	if err := b.ws.SetWriteDeadline(time.Now().Add(20 * time.Second)); err != nil {
		return nil, err
	}
	if err := b.ws.WriteJSON(msg); err != nil {
		return nil, err
	}
	want := b.nextID
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		_ = b.ws.SetReadDeadline(time.Now().Add(20 * time.Second))
		var resp map[string]any
		if err := b.ws.ReadJSON(&resp); err != nil {
			return nil, err
		}
		id, _ := resp["id"].(float64)
		if int(id) != want {
			continue
		}
		if errv, ok := resp["error"]; ok && errv != nil {
			return nil, fmt.Errorf("cdp %s: %v", method, errv)
		}
		result, _ := resp["result"].(map[string]any)
		if result == nil {
			result = map[string]any{}
		}
		return result, nil
	}
	return nil, fmt.Errorf("cdp timeout: %s", method)
}

func (b *Browser) evaluate(expression string) (any, error) {
	if b.rt != nil {
		return b.rt.Evaluate(expression)
	}
	return b.evaluateWS(expression)
}

func (b *Browser) evaluateWS(expression string) (any, error) {
	res, err := b.sessionCall("Runtime.evaluate", map[string]any{
		"expression": expression, "returnByValue": true,
	})
	if err != nil {
		return nil, err
	}
	if _, ok := res["exceptionDetails"]; ok {
		return nil, fmt.Errorf("page changed during evaluation")
	}
	result, _ := res["result"].(map[string]any)
	if result == nil {
		return nil, nil
	}
	return result["value"], nil
}

func (b *Browser) Observe(screenshot bool) (page.Page, error) {
	raw, err := b.evaluate(snapshotJS())
	if err != nil {
		return page.Page{}, err
	}
	if raw == nil {
		return page.Page{}, fmt.Errorf("document is navigating")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return page.Page{}, err
	}
	var p page.Page
	if err := json.Unmarshal(encoded, &p); err != nil {
		return page.Page{}, err
	}
	for i := range p.Actions {
		p.Actions[i].Scope = "web"
	}
	p = page.WithFingerprint(p)
	if screenshot {
		shot, err := b.inputCall("Page.captureScreenshot", map[string]any{"format": "jpeg", "quality": 72})
		if err == nil {
			p.Screenshot, _ = shot["data"].(string)
		}
	}
	return p, nil
}

func (b *Browser) Fresh(p page.Page, action *page.Action) bool {
	if action != nil && (action.Kind == "click" || action.Kind == "select" || action.Kind == "fill") {
		node, ok := action.Node.(float64)
		if !ok {
			if n, ok := action.Node.(int); ok {
				node = float64(n)
			} else {
				return false
			}
		}
		expr := fmt.Sprintf("(() => { const c=window.__jevium; return c ? [c.pageKey(),c.guard(c.nodes.get(%d))] : null; })()", int(node))
		cur, err := b.evaluate(expr)
		if err != nil {
			return false
		}
		want, _ := json.Marshal([]any{p.PageKey, guardFor(p, int(node))})
		got, _ := json.Marshal(cur)
		return string(want) == string(got)
	}
	cur, err := b.evaluate("(() => { const state=" + snapshotJS() + "; return state?.marker ?? null; })()")
	if err != nil {
		return false
	}
	want, _ := json.Marshal(p.Marker)
	got, _ := json.Marshal(cur)
	return string(want) == string(got)
}

func (b *Browser) nodePresent(action page.Action) bool {
	node, ok := action.Node.(float64)
	if !ok {
		if n, ok := action.Node.(int); ok {
			node = float64(n)
		} else {
			return false
		}
	}
	cur, err := b.evaluate(fmt.Sprintf("(() => { const e=window.__jevium?.nodes.get(%d); return !!(e&&e.isConnected); })()", int(node)))
	if err != nil {
		return false
	}
	present, _ := cur.(bool)
	return present
}

func guardFor(p page.Page, node int) any {
	if p.Guards == nil {
		return nil
	}
	return p.Guards[fmt.Sprintf("%d", node)]
}

func (b *Browser) inputCall(method string, params map[string]any) (map[string]any, error) {
	if b.rt != nil {
		return b.rt.Call(method, params, true)
	}
	return b.sessionCall(method, params)
}

func (b *Browser) Act(action page.Action, p page.Page, text *string) error {
	if !b.Fresh(p, &action) {
		return stale.Error{Msg: "page changed since this decision. Observe again"}
	}
	if action.Kind == "wait" {
		time.Sleep(100 * time.Millisecond)
		return nil
	}
	if action.Kind == "scroll" {
		_, err := b.inputCall("Input.dispatchMouseEvent", map[string]any{
			"type": "mouseWheel", "x": 550, "y": 650, "deltaX": 0, "deltaY": action.Delta,
		})
		return err
	}
	if action.Kind == "fill" && (text == nil || strings.TrimSpace(*text) == "") {
		return fmt.Errorf("TYPE_TEXT needs text from the helper; the executor does not guess")
	}
	encoded, _ := json.Marshal(action)
	targetExpr := `(action => {
      const e=window.__jevium?.nodes.get(action.node);
      if (!e?.isConnected || e.matches(':disabled') || e.closest('[aria-disabled="true"],[inert]') ||
          !e.checkVisibility({checkOpacity:true,checkVisibilityCSS:true})) return null;
      if (action.kind==='fill' && (e.readOnly || e.getAttribute('aria-readonly')==='true')) return null;
      const r=e.getBoundingClientRect(), x=r.x+r.width/2, y=r.y+r.height/2;
      if (!r.width || !r.height || x<0 || y<0 || x>=innerWidth || y>=innerHeight) return null;
      if (!e.contains(document.elementFromPoint(x,y))) return null;
      if (action.kind==='select') {
        if (e.tagName!=='SELECT' || ![...e.options].some(o=>o.value===action.value &&
            !o.disabled && !o.closest('optgroup[disabled]'))) return null;
        e.value=action.value;
        e.dispatchEvent(new Event('input',{bubbles:true}));
        e.dispatchEvent(new Event('change',{bubbles:true}));
      }
      return {x,y};
    })(` + string(encoded) + `)`
	target, err := b.evaluate(targetExpr)
	if err != nil {
		return err
	}
	if target == nil {
		return stale.Error{Msg: "target changed or is covered. Observe again"}
	}
	if action.Kind == "select" {
		return nil
	}
	m, _ := target.(map[string]any)
	x, y := num(m["x"]), num(m["y"])
	for _, ev := range []string{"mousePressed", "mouseReleased"} {
		if _, err := b.inputCall("Input.dispatchMouseEvent", map[string]any{
			"type": ev, "x": x, "y": y, "button": "left", "clickCount": 1,
		}); err != nil {
			return err
		}
	}
	if action.Kind == "fill" {
		if !b.nodePresent(action) {
			return fmt.Errorf("field %q is gone after tap; not retrying", action.Label)
		}
		mod := 2
		if runtime.GOOS == "darwin" {
			mod = 4
		}
		_, _ = b.inputCall("Input.dispatchKeyEvent", map[string]any{
			"type": "keyDown", "key": "a", "code": "KeyA", "modifiers": mod, "commands": []string{"selectAll"},
		})
		_, _ = b.inputCall("Input.dispatchKeyEvent", map[string]any{
			"type": "keyUp", "key": "a", "code": "KeyA", "modifiers": mod,
		})
		_, err := b.inputCall("Input.insertText", map[string]any{"text": *text})
		return err
	}
	return nil
}

func (b *Browser) Close() error {
	if b.ws == nil {
		return nil
	}
	if b.target != "" {
		_, _ = b.call("Target.closeTarget", map[string]any{"targetId": b.target})
		b.target = ""
	}
	return b.ws.Close()
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func num(v any) float64 {
	f, _ := v.(float64)
	return f
}
