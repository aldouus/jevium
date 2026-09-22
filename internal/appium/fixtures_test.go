package appium

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestFixtureRoundTripAndFailedWriteIsNotRetried(t *testing.T) {
	for _, fail := range []bool{false, true} {
		t.Run(map[bool]string{false: "round trip", true: "failed write"}[fail], func(t *testing.T) {
			local := filepath.Join(t.TempDir(), "sample.pdf")
			content := []byte("%PDF fixture bytes\x00")
			if err := os.WriteFile(local, content, 0600); err != nil {
				t.Fatal(err)
			}
			var calls []string
			var stored string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var value any
				if strings.HasSuffix(r.URL.Path, "/push_file") {
					calls = append(calls, "push")
					var body map[string]string
					if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
						t.Error(err)
					}
					if body["path"] != "@com.example.files:documents/sample.pdf" {
						t.Errorf("path=%q", body["path"])
					}
					stored = body["data"]
					if fail {
						value = map[string]string{"error": "unknown error", "message": "fixture write denied"}
					}
				}
				if strings.HasSuffix(r.URL.Path, "/pull_file") {
					calls = append(calls, "pull")
					value = stored
				}
				json.NewEncoder(w).Encode(map[string]any{"value": value})
			}))
			defer srv.Close()
			_, err := New(Config{URL: srv.URL, UDID: "test", SessionID: "session", HTTP: srv.Client(), Fixtures: []Fixture{{local, "@com.example.files:documents/sample.pdf"}}})
			if fail {
				if err == nil || err.Error() != "fixture push failed (not retried): fixture write denied" {
					t.Fatalf("error=%v", err)
				}
				if !reflect.DeepEqual(calls, []string{"push"}) {
					t.Fatalf("calls=%v", calls)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(calls, []string{"push", "pull"}) {
					t.Fatalf("calls=%v", calls)
				}
				if stored != base64.StdEncoding.EncodeToString(content) {
					t.Fatal("fixture bytes changed")
				}
			}
		})
	}
}

func TestRejectUnsafeFixtureBeforeOpeningFile(t *testing.T) {
	for _, remote := range []string{"/DCIM/photo.jpg", "@com.example.files:documents/../secret", "@com.example.files:documents//absolute", "@com.example.files:data/photo.jpg"} {
		_, err := loadFixtures([]Fixture{{"missing", remote}})
		if err == nil || err.Error() != "fixture destination must name a file inside @bundle.id:documents/" {
			t.Fatalf("destination %q error=%v", remote, err)
		}
	}
}

func TestFixtureVerificationMismatchDoesNotRepush(t *testing.T) {
	pushes := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var value any
		if strings.HasSuffix(r.URL.Path, "/push_file") {
			pushes++
		}
		if strings.HasSuffix(r.URL.Path, "/pull_file") {
			value = base64.StdEncoding.EncodeToString([]byte("wrong bytes"))
		}
		json.NewEncoder(w).Encode(map[string]any{"value": value})
	}))
	defer srv.Close()
	d := Device{cfg: Config{URL: srv.URL}, sessionID: "session", http: srv.Client()}
	err := d.provision([]loadedFixture{{Fixture: Fixture{RemotePath: "@com.example.files:documents/file"}, data: []byte("expected bytes")}})
	if err == nil || err.Error() != "fixture pushed but device bytes did not match local fixture" {
		t.Fatalf("error=%v", err)
	}
	if pushes != 1 {
		t.Fatalf("push count=%d", pushes)
	}
}
