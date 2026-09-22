package appium_test

import (
	"strings"
	"testing"

	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
)

const springboard = `<?xml version="1.0" encoding="UTF-8"?>
<AppiumAUT>
  <XCUIElementTypeApplication name="SpringBoard" bundleId="com.apple.springboard" visible="true">
    <XCUIElementTypeWindow visible="true" enabled="true">
      <XCUIElementTypeIcon name="Safari" label="Safari" type="XCUIElementTypeIcon"
        enabled="true" visible="true" accessible="true" x="24" y="80" width="60" height="60"/>
      <XCUIElementTypeIcon name="Settings" label="Settings" type="XCUIElementTypeIcon"
        enabled="true" visible="true" accessible="true" x="100" y="80" width="60" height="60"/>
      <XCUIElementTypeIcon name="Hidden" label="Hidden" type="XCUIElementTypeIcon"
        enabled="true" visible="false" accessible="true" x="176" y="80" width="60" height="60"/>
    </XCUIElementTypeWindow>
  </XCUIElementTypeApplication>
</AppiumAUT>`

const safariSearch = `<?xml version="1.0" encoding="UTF-8"?>
<AppiumAUT>
  <XCUIElementTypeApplication name="Safari" label="Safari" bundleId="com.apple.mobilesafari" visible="true">
    <XCUIElementTypeWindow visible="true" enabled="true" width="375" height="812">
      <XCUIElementTypeOther visible="true">
        <XCUIElementTypeStaticText value="Example Domain" label="Example Domain"
          visible="true" enabled="true" x="20" y="120" width="335" height="24"/>
        <XCUIElementTypeButton name="Tabs" label="Tabs" type="XCUIElementTypeButton"
          enabled="true" visible="true" accessible="true" x="320" y="54" width="40" height="32"/>
        <XCUIElementTypeTextField name="TabBarItemTitle" label="Address" value="example.com"
          type="XCUIElementTypeTextField" enabled="true" visible="true" accessible="true"
          x="48" y="54" width="240" height="32"/>
        <XCUIElementTypeSecureTextField name="Password" label="Password" value="••••"
          type="XCUIElementTypeSecureTextField" enabled="true" visible="true" accessible="true"
          x="48" y="400" width="240" height="32"/>
        <XCUIElementTypePickerWheel name="Month" label="Month" value="September"
          type="XCUIElementTypePickerWheel" enabled="true" visible="true" accessible="true"
          x="20" y="500" width="160" height="120"
          values="January,February,March,April,May,June,July,August,September,October,November,December"/>
        <XCUIElementTypeSwitch name="Private" label="Private" value="0"
          type="XCUIElementTypeSwitch" enabled="true" visible="true" accessible="true"
          x="20" y="640" width="80" height="32"/>
        <XCUIElementTypeAlert name="Allow Paste" label="Allow Paste" visible="true" enabled="true">
          <XCUIElementTypeButton name="Allow Paste" label="Allow Paste"
            enabled="true" visible="true" accessible="true" x="40" y="360" width="140" height="44"/>
          <XCUIElementTypeButton name="Don't Allow" label="Don't Allow"
            enabled="true" visible="true" accessible="true" x="196" y="360" width="140" height="44"/>
        </XCUIElementTypeAlert>
      </XCUIElementTypeOther>
    </XCUIElementTypeWindow>
  </XCUIElementTypeApplication>
</AppiumAUT>`

func kinds(p page.Page, kind string) []page.Action {
	var out []page.Action
	for _, a := range p.Actions {
		if a.Kind == kind {
			out = append(out, a)
		}
	}
	return out
}

func TestSpringBoardIconsBecomeClicks(t *testing.T) {
	t.Parallel()
	p, err := appium.SnapshotFromSource(springboard, "com.apple.springboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	clicks := map[string]page.Action{}
	for _, a := range kinds(p, "click") {
		clicks[a.Label] = a
	}
	if _, ok := clicks["Safari"]; !ok {
		t.Fatal("Safari")
	}
	if _, ok := clicks["Hidden"]; ok {
		t.Fatal("hidden icon offered")
	}
	ids := map[string]struct{}{}
	for _, a := range p.Actions {
		ids[a.ID] = struct{}{}
	}
	if _, ok := ids["wait"]; !ok {
		t.Fatal("wait")
	}
	if _, ok := ids["home"]; !ok {
		t.Fatal("home")
	}
	if p.URL != "app://com.apple.springboard" || p.Title != "SpringBoard" {
		t.Fatalf("page=%s %s", p.URL, p.Title)
	}
}

func TestHiddenAndSecureFieldsAreNotOffered(t *testing.T) {
	t.Parallel()
	p, err := appium.SnapshotFromSource(safariSearch, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range p.Actions {
		if a.Label == "Password" {
			t.Fatal("password offered")
		}
	}
}

func TestTextFieldsOfferFillAndOpen(t *testing.T) {
	t.Parallel()
	p, err := appium.SnapshotFromSource(safariSearch, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	var fill, open page.Action
	for _, a := range p.Actions {
		if a.Label == "Address" && a.Kind == "fill" {
			fill = a
		}
		if a.Label == "Open Address" && a.Kind == "click" {
			open = a
		}
	}
	if fill.ID == "" || open.ID == "" {
		t.Fatal("missing fill/open")
	}
	if fill.Node != open.Node || fill.Value != "example.com" || fill.Role != "textbox" {
		t.Fatalf("fill=%+v open=%+v", fill, open)
	}
}

func TestPickerWheelEmitsUnselectedOptions(t *testing.T) {
	t.Parallel()
	p, err := appium.SnapshotFromSource(safariSearch, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	values := map[string]struct{}{}
	for _, a := range kinds(p, "select") {
		values[a.Value] = struct{}{}
		if !strings.HasPrefix(a.Label, "Month → ") {
			t.Fatalf("label=%s", a.Label)
		}
	}
	if _, ok := values["September"]; ok {
		t.Fatal("selected option offered")
	}
	if _, ok := values["January"]; !ok {
		t.Fatal("January")
	}
}

func TestAlertsOfferControls(t *testing.T) {
	t.Parallel()
	p, err := appium.SnapshotFromSource(safariSearch, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	ids := map[string]struct{}{}
	labels := map[string]struct{}{}
	for _, a := range p.Actions {
		ids[a.ID] = struct{}{}
		if a.Kind == "click" {
			labels[a.Label] = struct{}{}
		}
	}
	if _, ok := ids["accept_alert"]; !ok {
		t.Fatal("accept_alert")
	}
	if _, ok := labels["Allow Paste"]; !ok {
		t.Fatal("Allow Paste")
	}
}

func TestVisibleStaticTextIsPageText(t *testing.T) {
	t.Parallel()
	p, err := appium.SnapshotFromSource(safariSearch, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(p.Text, "Example Domain") {
		t.Fatalf("text=%s", p.Text)
	}
	for _, a := range p.Actions {
		if a.Label == "Example Domain" {
			t.Fatal("static text became an action")
		}
	}
}

func TestActionCapKeepsControls(t *testing.T) {
	t.Parallel()
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><AppiumAUT><XCUIElementTypeApplication name="SpringBoard" bundleId="com.apple.springboard" visible="true"><XCUIElementTypeWindow visible="true" enabled="true">`)
	for i := range 400 {
		b.WriteString(`<XCUIElementTypeIcon name="App` + itoa(i) + `" label="App` + itoa(i) + `" enabled="true" visible="true" accessible="true" x="0" y="0" width="32" height="32"/>`)
	}
	b.WriteString(`</XCUIElementTypeWindow></XCUIElementTypeApplication></AppiumAUT>`)
	p, err := appium.SnapshotFromSource(b.String(), "com.apple.springboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	ids := map[string]struct{}{}
	for _, a := range p.Actions {
		ids[a.ID] = struct{}{}
		if strings.HasPrefix(a.ID, "e") {
			n++
		}
	}
	if n != page.MaxElementActions {
		t.Fatalf("element actions=%d", n)
	}
	if p.OmittedActions != 150 {
		t.Fatalf("omitted=%d", p.OmittedActions)
	}
	if _, ok := ids["wait"]; !ok {
		t.Fatal("wait dropped")
	}
}

func TestFingerprintIgnoresScreenshots(t *testing.T) {
	t.Parallel()
	p, err := appium.SnapshotFromSource(springboard, "com.apple.springboard", nil)
	if err != nil {
		t.Fatal(err)
	}
	other := p
	other.Screenshot = "changed"
	if page.Fingerprint(p) != page.Fingerprint(other) {
		t.Fatal("screenshot affected fingerprint")
	}
	other.Actions = append([]page.Action(nil), p.Actions...)
	other.Actions[0].Node = "other-node"
	if page.Fingerprint(p) == page.Fingerprint(other) {
		t.Fatal("node identity ignored")
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func TestWebViewOffersScrollOperations(t *testing.T) {
	p, err := appium.SnapshotFromSource(`<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeScrollView visible="true" x="0" y="50" width="375" height="650"><XCUIElementTypeWebView visible="true" x="0" y="50" width="375" height="650"/></XCUIElementTypeScrollView></XCUIElementTypeWindow></AppiumAUT>`, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	_, targets, _ := policy.ActionSpace(p.Actions)
	for operation, direction := range map[string]string{"SCROLL_DOWN": "up", "SCROLL_UP": "down"} {
		if len(targets[operation]) != 1 {
			t.Fatalf("%s targets=%v", operation, targets[operation])
		}
		a, ok := targets[operation]["1"]
		if !ok || a.Kind != "scroll" || a.Direction != direction {
			t.Fatalf("%s: action=%+v found=%v", operation, a, ok)
		}
	}
}

func TestAccessibleCustomControlIsOffered(t *testing.T) {
	p, err := appium.SnapshotFromSource(`<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeOther label="Menu" visible="true" accessible="true" hittable="true" x="271" y="114" width="80" height="44"/><XCUIElementTypeOther label="Layout" visible="true" accessible="false" hittable="true" x="0" y="50" width="375" height="650"/></XCUIElementTypeWindow></AppiumAUT>`, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	clicks := kinds(p, "click")
	if len(clicks) != 1 || clicks[0].Label != "Menu" {
		t.Fatalf("clicks=%+v", clicks)
	}
}

func TestScrollTargetsHittableForegroundContainer(t *testing.T) {
	source := `<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeScrollView visible="true" hittable="false" x="0" y="0" width="375" height="812"/><XCUIElementTypeScrollView visible="true" hittable="true" x="20" y="400" width="335" height="350"/></XCUIElementTypeWindow></AppiumAUT>`
	p, err := appium.SnapshotFromSource(source, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"scroll_down", "scroll_up"} {
		a, ok := page.FindAction(p.Actions, id)
		if !ok || a.Rect == nil || *a.Rect != (page.Rect{X: 20, Y: 400, W: 335, H: 350}) {
			t.Fatalf("%s: action=%+v found=%v", id, a, ok)
		}
	}
}

func TestSnapshotRejectsCoveredAndClippedControls(t *testing.T) {
	source := `<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeScrollView visible="true" x="0" y="50" width="375" height="550">
<XCUIElementTypeButton label="Available" visible="true" hittable="true" x="10" y="100" width="100" height="40"/>
<XCUIElementTypeButton label="Covered" visible="true" hittable="false" x="10" y="200" width="100" height="40"/>
<XCUIElementTypeButton label="Below container" visible="true" hittable="true" x="10" y="650" width="100" height="40"/>
<XCUIElementTypeStaticText label="Offscreen answer" visible="true" x="10" y="850" width="100" height="40"/>
</XCUIElementTypeScrollView></XCUIElementTypeWindow></AppiumAUT>`
	p, err := appium.SnapshotFromSource(source, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	clicks := kinds(p, "click")
	if len(clicks) != 1 || clicks[0].Label != "Available" {
		t.Fatalf("clicks=%+v", clicks)
	}
	if p.Text != "" {
		t.Fatalf("offscreen text exposed: %q", p.Text)
	}
}

func TestSnapshotIgnoresDecorativeSourceChangesButTracksTargetGeometry(t *testing.T) {
	source := `<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeOther x="0" y="0" width="375" height="812" animationTick="1"><XCUIElementTypeButton label="Menu" visible="true" x="10" y="100" width="100" height="40"/></XCUIElementTypeOther></XCUIElementTypeWindow></AppiumAUT>`
	snapshot := func(s string) page.Page {
		t.Helper()
		p, err := appium.SnapshotFromSource(s, "com.apple.mobilesafari", nil)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	original := snapshot(source)
	decorative := snapshot(strings.Replace(source, `animationTick="1"`, `animationTick="2"`, 1))
	if original.Fingerprint != decorative.Fingerprint {
		t.Fatal("decorative XML invalidates unchanged targets")
	}
	inserted := snapshot(strings.Replace(source, `<XCUIElementTypeButton`, `<XCUIElementTypeOther accessible="false" x="0" y="0" width="30" height="30"/><XCUIElementTypeButton`, 1))
	if original.Fingerprint != inserted.Fingerprint {
		t.Fatal("decorative sibling invalidates unchanged targets")
	}
	moved := snapshot(strings.Replace(source, `y="100"`, `y="120"`, 1))
	if original.Fingerprint == moved.Fingerprint {
		t.Fatal("moved tap target did not invalidate snapshot")
	}
}

func TestNativeTargetIdentityChangesFingerprint(t *testing.T) {
	source := `<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeButton name="approve-alice" label="Approve" visible="true" x="10" y="100" width="100" height="40"/></XCUIElementTypeWindow></AppiumAUT>`
	a, err := appium.SnapshotFromSource(source, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := appium.SnapshotFromSource(strings.Replace(source, "approve-alice", "approve-bob", 1), "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint == b.Fingerprint {
		t.Fatal("replacement target was treated as fresh")
	}
}

func TestVisibleWebContentSurvivesOffscreenStructuralAncestor(t *testing.T) {
	source := `<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeWebView visible="true" x="0" y="0" width="375" height="812"><XCUIElementTypeOther visible="false" x="0" y="-1000" width="375" height="812"><XCUIElementTypeStaticText label="Visible paragraph" visible="true" x="16" y="100" width="300" height="40"/><XCUIElementTypeButton label="Visible button" visible="true" hittable="true" x="16" y="160" width="100" height="40"/></XCUIElementTypeOther></XCUIElementTypeWebView></XCUIElementTypeWindow></AppiumAUT>`
	p, err := appium.SnapshotFromSource(source, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if p.Text != "Visible paragraph" {
		t.Fatalf("text=%q", p.Text)
	}
	clicks := kinds(p, "click")
	if len(clicks) != 1 || clicks[0].Label != "Visible button" {
		t.Fatalf("clicks=%+v", clicks)
	}
}
