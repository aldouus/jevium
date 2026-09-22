package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/page"
	"testing"
)

func TestConfiguredNavigation(t *testing.T) {
	srv, calls := mockAppium(t, oneIcon, nil)
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", BundleID: "com.apple.mobilesafari", StartURL: "https://example.test/path", AllowedApps: []string{"com.example.app"}, HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"terminate_app_0", "activate_app_0"} {
		a, ok := page.FindAction(p.Actions, id)
		if !ok {
			t.Fatalf("missing %s", id)
		}
		if err := d.Act(a, p, nil); err != nil {
			t.Fatal(err)
		}
	}
	var scripts []string
	for _, c := range *calls {
		if s, ok := c.Body["script"].(string); ok {
			scripts = append(scripts, s)
		}
	}
	want := []string{"mobile: deepLink", "mobile: terminateApp", "mobile: activateApp"}
	if len(scripts) != len(want) {
		t.Fatalf("scripts=%v", scripts)
	}
	for i := range want {
		if scripts[i] != want[i] {
			t.Fatalf("scripts=%v", scripts)
		}
	}
	a, _ := page.FindAction(p.Actions, "activate_app_0")
	a.Value = "com.unconfigured.app"
	before := len(*calls)
	if err := d.Act(a, p, nil); err == nil {
		t.Fatal("unconfigured app accepted")
	}
	if len(*calls) != before+1 {
		t.Fatal("unexpected mutation")
	}
}

func TestNavigationValidationBeforeConnecting(t *testing.T) {
	for _, u := range []string{"file:///tmp/a", "https://user:pass@example.test", "relative"} {
		_, err := appium.New(appium.Config{StartURL: u, BundleID: "com.apple.mobilesafari"})
		if err == nil {
			t.Fatalf("accepted %q", u)
		}
	}
}
