package appium_test

import (
	"encoding/json"
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/policy"
	"strings"
	"testing"
)

func TestConfiguredSecretNeverEntersObservation(t *testing.T) {
	t.Setenv("TEST_PASSWORD", "private-password")
	const source = `<AppiumAUT>
  <XCUIElementTypeSecureTextField label="Password" value="private-password"
    x="20" y="30" width="200" height="40"/>
  <XCUIElementTypeStaticText label="private-password" x="0" y="80" width="100" height="20"/>
</AppiumAUT>`
	srv, _ := mockAppium(t, source, nil)
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", HTTP: srv.Client(), SecretFields: map[string]string{"Password": "TEST_PASSWORD"}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(true)
	if err != nil {
		t.Fatal(err)
	}
	_, targets, _ := policy.ActionSpace(p.Actions)
	if len(targets["TYPE_SECRET"]) != 1 {
		t.Fatalf("secret targets=%v", targets)
	}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "private-password") {
		t.Fatal("secret leaked into observation")
	}
	if p.Source != "" || p.Screenshot != "" {
		t.Fatal("unredacted capture retained")
	}
}

func TestUnconfiguredSecureFieldIsNotOffered(t *testing.T) {
	const source = `<AppiumAUT>
  <XCUIElementTypeSecureTextField label="Password" value="private-password"
    x="20" y="30" width="200" height="40"/>
</AppiumAUT>`
	srv, _ := mockAppium(t, source, nil)
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", HTTP: srv.Client()})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	_, targets, _ := policy.ActionSpace(p.Actions)
	if len(targets["TYPE_SECRET"]) != 0 {
		t.Fatalf("unconfigured targets=%v", targets)
	}
}
