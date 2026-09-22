package appium_test

import (
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/page"
	"strings"
	"testing"
)

func TestConfiguredNavigation(t *testing.T) {
	active := "com.apple.mobilesafari"
	srv, calls := mockAppium(t, strings.ReplaceAll(oneIcon, `bundleId="com.apple.springboard"`, ``), func(r *recorded) any {
		if r.Body["script"] == "mobile: activeAppInfo" {
			return map[string]string{"bundleId": active}
		}
		if r.Body["script"] == "mobile: activateApp" {
			active = "com.example.app"
		}
		return nil
	})
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
		if c.Body["script"] == "mobile: deepLink" {
			args := c.Body["args"].([]any)
			if len(args) != 1 {
				t.Fatalf("deep link args=%v", args)
			}
			arg := args[0].(map[string]any)
			if arg["url"] != "https://example.test/path" || arg["bundleId"] != "com.apple.mobilesafari" {
				t.Fatalf("deep link=%v", arg)
			}
		}
		if s, ok := c.Body["script"].(string); ok && s != "mobile: activeAppInfo" {
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
	p, err = d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	if p.BundleID != "com.example.app" {
		t.Fatalf("foreground bundle=%q", p.BundleID)
	}
	a, _ := page.FindAction(p.Actions, "activate_app_0")
	a.Value = "com.unconfigured.app"
	before := len(*calls)
	if err := d.Act(a, p, nil); err == nil || err.Error() != "App is not configured for navigation" {
		t.Fatalf("unconfigured app error=%v", err)
	}
	for _, c := range (*calls)[before:] {
		if c.Method == "POST" && c.Body["script"] != "mobile: activeAppInfo" {
			t.Fatalf("unexpected mutation %+v", c)
		}
	}
}

func TestNavigationValidationBeforeConnecting(t *testing.T) {
	for _, u := range []string{"file:///tmp/a", "https://user:pass@example.test", "relative"} {
		_, err := appium.New(appium.Config{StartURL: u, BundleID: "com.apple.mobilesafari"})
		if err == nil || err.Error() != "--url must be an absolute HTTP(S) URL without credentials" {
			t.Fatalf("url %q error=%v", u, err)
		}
	}
}
