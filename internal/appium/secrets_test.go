package appium_test

import (
	"encoding/json"
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/policy"
	"net/http"
	"net/http/httptest"
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

func TestSecretEntryVerifiesFocusAndRedactsDriverErrors(t *testing.T) {
	t.Setenv("TEST_PASSWORD", "private-password")
	const source = `<AppiumAUT>
  <XCUIElementTypeSecureTextField name="Password" label="Password"
    x="20.5" y="30" width="200" height="40"/>
</AppiumAUT>`
	for _, active := range []string{"field", "other", "after-clear"} {
		t.Run(active, func(t *testing.T) {
			cleared, typed := 0, 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var value any
				switch {
				case r.URL.Path == "/session":
					value = map[string]string{"sessionId": "test"}
				case strings.HasSuffix(r.URL.Path, "/source"):
					value = source
				case strings.HasSuffix(r.URL.Path, "/elements"):
					value = []map[string]string{{"element-6066-11e4-a52e-4f735466cecf": "field"}}
				case strings.HasSuffix(r.URL.Path, "/attribute/name"), strings.HasSuffix(r.URL.Path, "/attribute/label"):
					value = "Password"
				case strings.HasSuffix(r.URL.Path, "/attribute/type"):
					value = "XCUIElementTypeSecureTextField"
				case strings.HasSuffix(r.URL.Path, "/rect"):
					value = map[string]float64{"x": 20.5, "y": 30, "width": 200, "height": 40}
				case strings.HasSuffix(r.URL.Path, "/element/active"):
					focused := active
					if active == "after-clear" {
						focused = "field"
						if cleared > 0 {
							focused = "other"
						}
					}
					value = map[string]string{"element-6066-11e4-a52e-4f735466cecf": focused}
				case strings.HasSuffix(r.URL.Path, "/clear"):
					cleared++
				case strings.HasSuffix(r.URL.Path, "/actions"):
					typed++
					value = map[string]string{"error": "unknown error", "message": "private-password failed"}
				}
				json.NewEncoder(w).Encode(map[string]any{"value": value})
			}))
			defer srv.Close()
			d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", HTTP: srv.Client(), SecretFields: map[string]string{"Password": "TEST_PASSWORD"}})
			if err != nil {
				t.Fatal(err)
			}
			p, err := d.Observe(false)
			if err != nil {
				t.Fatal(err)
			}
			_, targets, _ := policy.ActionSpace(p.Actions)
			if len(targets["TYPE_SECRET"]) != 1 {
				t.Fatalf("targets=%v", targets)
			}
			for _, a := range targets["TYPE_SECRET"] {
				err = d.Act(a, p, nil)
			}
			if err == nil || strings.Contains(err.Error(), "private-password") {
				t.Fatalf("error leaked or missing: %v", err)
			}
			want := 0
			if active == "field" {
				want = 1
			}
			wantClear := want
			if active == "after-clear" {
				wantClear = 1
			}
			if cleared != wantClear || typed != want {
				t.Fatalf("cleared=%d typed=%d want=%d", cleared, typed, want)
			}
		})
	}
}

func TestDuplicateSecretLabelsAreExcluded(t *testing.T) {
	t.Setenv("TEST_PASSWORD", "private-password")
	const source = `<AppiumAUT>
  <XCUIElementTypeSecureTextField label="Password" x="20" y="30" width="200" height="40"/>
  <XCUIElementTypeSecureTextField label="Password" x="20" y="90" width="200" height="40"/>
</AppiumAUT>`
	srv, _ := mockAppium(t, source, nil)
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", HTTP: srv.Client(), SecretFields: map[string]string{"Password": "TEST_PASSWORD"}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	_, targets, _ := policy.ActionSpace(p.Actions)
	if len(targets["TYPE_SECRET"]) != 0 {
		t.Fatalf("ambiguous targets=%v", targets)
	}
}

func TestSecretIsRedactedBeforeTextTruncation(t *testing.T) {
	t.Setenv("TEST_PASSWORD", "private-password-value")
	source := `<AppiumAUT><XCUIElementTypeStaticText x="0" y="0" width="200" height="40" label="` + strings.Repeat("x", 5990) + `private-password-value"/></AppiumAUT>`
	srv, _ := mockAppium(t, source, nil)
	d, err := appium.New(appium.Config{URL: srv.URL, UDID: "test", HTTP: srv.Client(), SecretFields: map[string]string{"Password": "TEST_PASSWORD"}})
	if err != nil {
		t.Fatal(err)
	}
	p, err := d.Observe(false)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.Text, "private") {
		t.Fatal("partial secret leaked at text boundary")
	}
	if p.Text != strings.Repeat("x", 5990)+"[redacted]" {
		t.Fatalf("redacted text length=%d", len(p.Text))
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
