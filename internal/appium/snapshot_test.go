package appium_test

import (
	"strings"
	"testing"

	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/page"
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
