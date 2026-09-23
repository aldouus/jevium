package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestDeviceJobsRejectCollisions(t *testing.T) {
	for _, tc := range []struct {
		ids        string
		wda, mjpeg int
	}{{"A,A", 8100, 9100}, {"../A", 8100, 9100}, {"A,B", 8100, 8101}, {"A,B", 65535, 9100}, {"A,", 8100, 9100}} {
		if _, err := deviceJobs(tc.ids, t.TempDir(), "", tc.wda, tc.mjpeg); err == nil {
			t.Fatalf("accepted invalid plan %+v", tc)
		}
	}
}

func TestDevicesRunConcurrentlyAndKeepIndependentResults(t *testing.T) {
	root := t.TempDir()
	jobs, err := deviceJobs("phone-a,phone-b", root, "", 8100, 9100)
	if err != nil {
		t.Fatal(err)
	}
	for i := range jobs {
		jobs[i].Coverage = filepath.Join(root, jobs[i].UDID+".json")
	}
	expectations := []string{`{"field":"title","value":"Done"}`, `{"field":"text","value":"Complete"}`}
	auditURLs := []string{"https://example.test/one", "https://example.test/two"}
	arrived := make(chan struct{}, 2)
	release := make(chan struct{})
	var mu sync.Mutex
	calls := map[string]int{}
	launch := func(_ string, args []string, log io.Writer) error {
		flags := map[string]string{}
		repeated := map[string][]string{}
		for i := 0; i < len(args); i += 2 {
			flags[args[i]] = args[i+1]
			repeated[args[i]] = append(repeated[args[i]], args[i+1])
		}
		id := flags["--udid"]
		mu.Lock()
		calls[id]++
		mu.Unlock()
		arrived <- struct{}{}
		<-release
		if flags["--session-id"] != "" {
			return fmt.Errorf("inherited session")
		}
		if flags["--scope"] != "web" || !reflect.DeepEqual(repeated["--expect"], expectations) || !reflect.DeepEqual(repeated["--audit-url"], auditURLs) {
			return fmt.Errorf("lost audit arguments: %v", repeated)
		}
		if flags["--coverage"] != filepath.Join(root, id+".json") {
			return fmt.Errorf("shared coverage output: %q", flags["--coverage"])
		}
		if err := os.WriteFile(flags["--coverage"], []byte(id), 0600); err != nil {
			return err
		}
		i := 0
		if id == "phone-b" {
			i = 1
		}
		if flags["--wda-local-port"] != fmt.Sprint(8100+i) || flags["--mjpeg-server-port"] != fmt.Sprint(9100+i) || flags["--derived-data-path"] != filepath.Join(root, "derived-data", id) {
			return fmt.Errorf("wrong isolation flags %v", flags)
		}
		if _, err := fmt.Fprintln(log, id); err != nil {
			return err
		}
		status := "done"
		if i == 1 {
			status = "blocked"
		}
		return os.WriteFile(flags["--result-file"], []byte(fmt.Sprintf(`{"status":%q}`, status)), 0600)
	}
	var output bytes.Buffer
	finished := make(chan error, 1)
	go func() {
		finished <- runDevices(jobs, "unused", "server", "bundle", []string{"goal"}, &output, launch, "--scope", "web", "--expect", expectations[0], "--expect", expectations[1], "--audit-url", auditURLs[0], "--audit-url", auditURLs[1])
	}()
	<-arrived
	<-arrived
	close(release)
	if err := <-finished; err == nil {
		t.Fatal("blocked device reported successful aggregate")
	}
	decoder := json.NewDecoder(&output)
	for i, job := range jobs {
		var result deviceResult
		if err := decoder.Decode(&result); err != nil {
			t.Fatal(err)
		}
		want := "done"
		if i == 1 {
			want = "blocked"
		}
		if result.UDID != job.UDID || result.Status != want || calls[job.UDID] != 1 {
			t.Fatalf("result=%+v calls=%v", result, calls)
		}
		data, err := os.ReadFile(filepath.Join(job.Directory, "actions.log"))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != job.UDID+"\n" {
			t.Fatalf("mixed log %q", data)
		}
		coverage, err := os.ReadFile(job.Coverage)
		if err != nil {
			t.Fatal(err)
		}
		if string(coverage) != job.UDID {
			t.Fatalf("mixed coverage %q", coverage)
		}
	}
}

func TestDevicesNeverReplayFailedRun(t *testing.T) {
	jobs, err := deviceJobs("phone", t.TempDir(), "", 8100, 9100)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	launch := func(string, []string, io.Writer) error { calls++; return fmt.Errorf("connection lost after tap") }
	if err := runDevices(jobs, "", "", "", nil, io.Discard, launch); err == nil {
		t.Fatal("lost failure")
	}
	if calls != 1 {
		t.Fatalf("replayed task %d times", calls)
	}
	if err := runDevices(jobs, "", "", "", nil, io.Discard, launch); err == nil {
		t.Fatal("overwrote existing evidence")
	}
	if calls != 1 {
		t.Fatalf("launched despite existing directory: %d", calls)
	}
}
