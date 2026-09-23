package appium

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVisualOCRRejectsSecretFieldsBeforeConnecting(t *testing.T) {
	t.Setenv("APPIUM_SECRET_FIELDS", `{"Password":"DEVICE_PASSWORD"}`)
	t.Setenv("DEVICE_PASSWORD", "private-value")
	_, err := New(Config{Visual: fixedOCR{}})
	if err == nil || err.Error() != "visual OCR cannot be combined with secret fields" {
		t.Fatalf("error=%v", err)
	}
}

func TestInvisibleKeyboardDoesNotRequestActiveElement(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		json.NewEncoder(w).Encode(map[string]any{"value": nil})
	}))
	defer srv.Close()
	d := Device{cfg: Config{URL: srv.URL}, http: srv.Client()}
	_, err := d.snapshot(`<AppiumAUT><XCUIElementTypeWindow width="375" height="812"><XCUIElementTypeKeyboard visible="false"/></XCUIElementTypeWindow></AppiumAUT>`)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatalf("hidden keyboard caused %d Appium calls", calls)
	}
}
