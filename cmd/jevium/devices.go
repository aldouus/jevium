package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type deviceJob struct {
	UDID, Directory, Derived string
	Coverage                 string
	WDA, MJPEG               int
}

func deviceJobs(ids, root, derived string, wda, mjpeg int) ([]deviceJob, error) {
	if root == "" {
		return nil, fmt.Errorf("--devices requires --record-dir")
	}
	if derived == "" {
		derived = filepath.Join(root, "derived-data")
	}
	seen := map[string]bool{}
	ports := map[int]bool{}
	var jobs []deviceJob
	for i, raw := range strings.Split(ids, ",") {
		id := strings.TrimSpace(raw)
		if !regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`).MatchString(id) || seen[id] {
			return nil, fmt.Errorf("invalid or duplicate device UDID %q", id)
		}
		seen[id] = true
		for _, port := range []int{wda + i, mjpeg + i} {
			if port < 1 || port > 65535 || ports[port] {
				return nil, fmt.Errorf("invalid or colliding device port %d", port)
			}
			ports[port] = true
		}
		jobs = append(jobs, deviceJob{UDID: id, Directory: filepath.Join(root, id), Derived: filepath.Join(derived, id), WDA: wda + i, MJPEG: mjpeg + i})
	}
	return jobs, nil
}

func listDevices(out io.Writer) error {
	cmd := exec.Command("xcrun", "devicectl", "list", "devices")
	cmd.Stdout, cmd.Stderr = out, out
	return cmd.Run()
}

type deviceProcess func(executable string, args []string, log io.Writer) error

func runDeviceProcess(executable string, args []string, log io.Writer) error {
	cmd := exec.Command(executable, args...)
	cmd.Stdout, cmd.Stderr = log, log
	return cmd.Run()
}

type deviceResult struct {
	UDID      string `json:"udid"`
	Status    string `json:"status"`
	Error     string `json:"error,omitempty"`
	Directory string `json:"directory"`
}

func runDevices(jobs []deviceJob, executable, server, bundle string, goals []string, out io.Writer, launch deviceProcess, childArgs ...string) error {
	for _, job := range jobs {
		if err := os.MkdirAll(filepath.Dir(job.Directory), 0700); err != nil {
			return err
		}
		if err := os.Mkdir(job.Directory, 0700); err != nil {
			return fmt.Errorf("reserve %s: %w", job.Directory, err)
		}
		if job.Coverage != "" {
			file, err := os.OpenFile(job.Coverage, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				return fmt.Errorf("reserve coverage %s: %w", job.Coverage, err)
			}
			if err := file.Close(); err != nil {
				return err
			}
		}
	}
	results := make([]deviceResult, len(jobs))
	var wg sync.WaitGroup
	for i, job := range jobs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := deviceResult{UDID: job.UDID, Directory: job.Directory, Status: "error"}
			defer func() { results[i] = result }()
			log, err := os.OpenFile(filepath.Join(job.Directory, "actions.log"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
			if err != nil {
				result.Error = err.Error()
				return
			}
			statePath := filepath.Join(job.Directory, "state.json")
			args := []string{"--mode", "appium", "--udid", job.UDID, "--session-id", "", "--appium-url", server, "--bundle-id", bundle, "--wda-local-port", fmt.Sprint(job.WDA), "--mjpeg-server-port", fmt.Sprint(job.MJPEG), "--derived-data-path", job.Derived, "--record-dir", job.Directory, "--result-file", statePath}
			args = append(args, childArgs...)
			if job.Coverage != "" {
				args = append(args, "--coverage", job.Coverage)
			}
			for _, goal := range goals {
				args = append(args, "--goal", goal)
			}
			err = launch(executable, args, log)
			err = errors.Join(err, log.Close())
			if err != nil {
				result.Error = err.Error()
				return
			}
			data, err := os.ReadFile(statePath)
			var state struct {
				Status string `json:"status"`
			}
			if err == nil {
				err = json.Unmarshal(data, &state)
			}
			if err != nil {
				result.Error = err.Error()
				return
			}
			if state.Status != "done" && state.Status != "blocked" {
				result.Error = "child returned without a terminal state"
				return
			}
			result.Status = state.Status
		}()
	}
	wg.Wait()
	var failure error
	for _, result := range results {
		if err := json.NewEncoder(out).Encode(result); err != nil {
			failure = errors.Join(failure, err)
		}
		if result.Status != "done" {
			failure = errors.Join(failure, fmt.Errorf("device %s: %s %s", result.UDID, result.Status, result.Error))
		}
	}
	return failure
}
