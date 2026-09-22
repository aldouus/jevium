package appium

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRetrieveSavesExactDeviceBytes(t *testing.T) {
	want := []byte("%PDF-1.7\x00\xffdownloaded artifact")
	destination := filepath.Join(t.TempDir(), "download.pdf")
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "POST" || r.URL.Path != "/session/session/appium/device/pull_file" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["path"] != "@com.example.files:documents/report.pdf" {
			t.Errorf("path=%q", body["path"])
		}
		json.NewEncoder(w).Encode(map[string]string{"value": base64.StdEncoding.EncodeToString(want)})
	}))
	defer srv.Close()
	d := Device{cfg: Config{URL: srv.URL, Retrievals: []Retrieval{{"@com.example.files:documents/report.pdf", destination}}}, sessionID: "session", http: srv.Client()}
	if err := d.RetrieveArtifacts(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("saved bytes=%v want=%v", got, want)
	}
	if calls != 1 {
		t.Fatalf("pull calls=%d", calls)
	}
	if err := d.RetrieveArtifacts(); err == nil || !strings.HasPrefix(err.Error(), "retrieval destination already exists:") {
		t.Fatalf("overwrite error=%v", err)
	}
	if calls != 1 {
		t.Fatal("pulled despite existing destination")
	}
}

func TestRetrieveRejectsFailedOrMalformedResponses(t *testing.T) {
	for _, tc := range []struct {
		name    string
		value   any
		message string
	}{
		{"device failure", map[string]string{"error": "unknown error", "message": "device disconnected"}, "device disconnected"},
		{"missing data", nil, "artifact response contained no file data"},
		{"malformed base64", "%%%", "artifact response is not valid base64:"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "download.pdf")
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				json.NewEncoder(w).Encode(map[string]any{"value": tc.value})
			}))
			defer srv.Close()
			d := Device{cfg: Config{URL: srv.URL, Retrievals: []Retrieval{{"@com.example.files:documents/report.pdf", destination}}}, sessionID: "session", http: srv.Client()}
			if err := d.RetrieveArtifacts(); err == nil || !strings.Contains(err.Error(), tc.message) {
				t.Fatalf("error=%v", err)
			}
			if _, err := os.Stat(destination); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("failed retrieval left destination: %v", err)
			}
			if calls != 1 {
				t.Fatalf("pull calls=%d", calls)
			}
		})
	}
}

func TestArtifactPublishNeverOverwritesRacingDestination(t *testing.T) {
	directory := t.TempDir()
	destination := filepath.Join(directory, "download.pdf")
	if err := os.WriteFile(destination, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := saveArtifact(destination, []byte("replace")); err == nil || !errors.Is(err, os.ErrExist) {
		t.Fatalf("publish error=%v", err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "keep" {
		t.Fatalf("existing data=%q", got)
	}
	files, err := os.ReadDir(directory)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("temporary files leaked: %v", files)
	}
}

func TestRetrievalValidationPrecedesConnection(t *testing.T) {
	_, err := New(Config{Retrievals: []Retrieval{{"@com.example.files:documents/../private", "file"}}})
	if err == nil || err.Error() != "retrieval source must name a file inside @bundle.id:documents/" {
		t.Fatalf("validation error=%v", err)
	}
}

func TestArtifactResponseIsBounded(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(strings.Repeat("x", 65))) }))
	defer srv.Close()
	d := Device{cfg: Config{URL: srv.URL}, http: srv.Client()}
	err := d.callLimited("POST", "/artifact", nil, nil, 64)
	if err == nil || err.Error() != "Appium response exceeds artifact size limit" {
		t.Fatalf("error=%v", err)
	}
}
