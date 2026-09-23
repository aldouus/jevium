package appium_test

import (
	"strings"
	"testing"

	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
)

var springboard = screen(fixtureNode{Kind: "Application", Name: "SpringBoard", BundleID: "com.apple.springboard", Children: []fixtureNode{
	window(0, 0,
		fixtureNode{Kind: "Icon", Name: "Safari", Label: "Safari", Accessible: true, Rect: bounds(24, 80, 60, 60)},
		fixtureNode{Kind: "Icon", Name: "Settings", Label: "Settings", Accessible: true, Rect: bounds(100, 80, 60, 60)},
		fixtureNode{Kind: "Icon", Name: "Hidden", Label: "Hidden", Accessible: true, Hidden: true, Rect: bounds(176, 80, 60, 60)},
	),
}})

var safariSearch = screen(fixtureNode{Kind: "Application", Name: "Safari", BundleID: "com.apple.mobilesafari", Children: []fixtureNode{
	window(375, 812,
		staticText("Example Domain", bounds(20, 120, 335, 24)),
		button("Tabs", bounds(320, 54, 40, 32)),
		fixtureNode{Kind: "TextField", Name: "TabBarItemTitle", Label: "Address", Value: "example.com", Rect: bounds(48, 54, 240, 32)},
		fixtureNode{Kind: "SecureTextField", Name: "Password", Label: "Password", Value: "••••", Rect: bounds(48, 400, 240, 32)},
		fixtureNode{Kind: "PickerWheel", Name: "Month", Label: "Month", Value: "September", Rect: bounds(20, 500, 160, 120), Values: "January,February,March,April,May,June,July,August,September,October,November,December"},
		fixtureNode{Kind: "Switch", Name: "Private", Label: "Private", Value: "0", Rect: bounds(20, 640, 80, 32)},
		fixtureNode{Kind: "Alert", Name: "Allow Paste", Label: "Allow Paste", Children: []fixtureNode{
			button("Allow Paste", bounds(40, 360, 140, 44)), button("Don't Allow", bounds(196, 360, 140, 44)),
		}},
	),
}})

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
	var icons []fixtureNode
	for i := range 400 {
		icons = append(icons, fixtureNode{Kind: "Icon", Name: "App" + itoa(i), Label: "App" + itoa(i), Accessible: true, Rect: bounds(0, 0, 32, 32)})
	}
	p, err := appium.SnapshotFromSource(screen(window(0, 0, icons...)), "com.apple.springboard", nil)
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
	source := screen(window(375, 812, scroll("", bounds(0, 50, 375, 650), fixtureNode{Kind: "WebView", Rect: bounds(0, 50, 375, 650)})))
	p, err := appium.SnapshotFromSource(source, "com.apple.mobilesafari", nil)
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
	source := screen(window(375, 812, fixtureNode{Kind: "Other", Label: "Menu", Accessible: true, Rect: bounds(271, 114, 80, 44)}, fixtureNode{Kind: "Other", Label: "Layout", Rect: bounds(0, 50, 375, 650)}))
	p, err := appium.SnapshotFromSource(source, "com.apple.mobilesafari", nil)
	if err != nil {
		t.Fatal(err)
	}
	clicks := kinds(p, "click")
	if len(clicks) != 1 || clicks[0].Label != "Menu" {
		t.Fatalf("clicks=%+v", clicks)
	}
}

func TestScrollTargetsHittableForegroundContainer(t *testing.T) {
	covered := scroll("", bounds(0, 0, 375, 812))
	covered.NotHittable = true
	source := screen(window(375, 812, covered, scroll("", bounds(20, 400, 335, 350))))
	p, err := appium.SnapshotFromSource(source, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"scroll_down", "scroll_up"} {
		a, ok := page.FindAction(p.Actions, id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if a.Rect == nil {
			t.Fatalf("%s has no rectangle", id)
		}
		want := bounds(20, 400, 335, 350)
		if *a.Rect != want {
			t.Fatalf("%s rectangle=%+v want=%+v", id, *a.Rect, want)
		}
	}
}

func TestSnapshotRejectsCoveredAndClippedControls(t *testing.T) {
	covered := button("Covered", bounds(10, 200, 100, 40))
	covered.NotHittable = true
	source := screen(window(375, 812, scroll("", bounds(0, 50, 375, 550), button("Available", bounds(10, 100, 100, 40)), covered, button("Below container", bounds(10, 650, 100, 40)), staticText("Offscreen answer", bounds(10, 850, 100, 40)))))
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
	decoration := fixtureNode{Kind: "Other", Rect: bounds(0, 0, 375, 812), AnimationTick: "1", Children: []fixtureNode{button("Menu", bounds(10, 100, 100, 40))}}
	snapshot := func(s string) page.Page {
		t.Helper()
		p, err := appium.SnapshotFromSource(s, "com.apple.mobilesafari", nil)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	original := snapshot(screen(window(375, 812, decoration)))
	decoration.AnimationTick = "2"
	decorative := snapshot(screen(window(375, 812, decoration)))
	if original.Fingerprint != decorative.Fingerprint {
		t.Fatal("decorative XML invalidates unchanged targets")
	}
	decoration.Children = append([]fixtureNode{{Kind: "Other", Rect: bounds(0, 0, 30, 30)}}, decoration.Children...)
	inserted := snapshot(screen(window(375, 812, decoration)))
	if original.Fingerprint != inserted.Fingerprint {
		t.Fatal("decorative sibling invalidates unchanged targets")
	}
	decoration.Children[1].Rect.Y = 120
	moved := snapshot(screen(window(375, 812, decoration)))
	if original.Fingerprint == moved.Fingerprint {
		t.Fatal("moved tap target did not invalidate snapshot")
	}
}

func TestNativeTargetIdentityChangesFingerprint(t *testing.T) {
	target := button("Approve", bounds(10, 100, 100, 40))
	target.Name = "approve-alice"
	source := screen(window(375, 812, target))
	a, err := appium.SnapshotFromSource(source, "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	target.Name = "approve-bob"
	b, err := appium.SnapshotFromSource(screen(window(375, 812, target)), "app", nil)
	if err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint == b.Fingerprint {
		t.Fatal("replacement target was treated as fresh")
	}
}

func TestVisibleWebContentSurvivesOffscreenStructuralAncestor(t *testing.T) {
	source := screen(window(375, 812, fixtureNode{Kind: "WebView", Rect: bounds(0, 0, 375, 812), Children: []fixtureNode{
		{Kind: "Other", Hidden: true, Rect: bounds(0, -1000, 375, 812), Children: []fixtureNode{staticText("Visible paragraph", bounds(16, 100, 300, 40)), button("Visible button", bounds(16, 160, 100, 40))}},
	}}))
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
