package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectCollidingOutputsBeforeStartingDevice(t *testing.T) {
	for _, name := range []string{"result and retrieval", "coverage and retrieval", "result and coverage", "recording", "multi-device"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			output := filepath.Join(root, "output.json")
			record := filepath.Join(root, "recordings")
			args := []string{"--retrieve", "@com.example.app:documents/report.json=" + output}
			switch name {
			case "result and retrieval":
				args = append(args, "--result-file", output)
			case "coverage and retrieval":
				args = append(args, "--coverage", output)
			case "result and coverage":
				args = []string{"--result-file", output, "--coverage", output}
			case "recording":
				args = []string{"--record-dir", record, "--retrieve", "@com.example.app:documents/photo.jpg=" + filepath.Join(record, "000000.jpg")}
			case "multi-device":
				args = append(args, "--devices", "one,two", "--record-dir", record, "--coverage", output, "--goal", "inspect")
			}
			err := run(args)
			if err == nil || !strings.Contains(err.Error(), "output path collision") {
				t.Fatalf("error=%v", err)
			}
			if _, err := os.Stat(record); !os.IsNotExist(err) {
				t.Fatalf("recording directory was created before rejecting output collision: %v", err)
			}
		})
	}
}

func TestOutputCollisionResolvesParentSymlinks(t *testing.T) {
	root := t.TempDir()
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(root, alias); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"--result-file", filepath.Join(root, "report.json"), "--retrieve", "@com.example.app:documents/report.json=" + filepath.Join(alias, "report.json")})
	if err == nil || !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("error=%v", err)
	}
}

func TestOutputCollisionResolvesDanglingResultSymlink(t *testing.T) {
	root := t.TempDir()
	result := filepath.Join(root, "state.json")
	artifact := filepath.Join(root, "artifact.json")
	if err := os.Symlink("artifact.json", result); err != nil {
		t.Fatal(err)
	}
	err := run([]string{"--result-file", result, "--retrieve", "@com.example.app:documents/report.json=" + artifact})
	if err == nil || !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("error=%v", err)
	}
}

func TestOutputCollisionRejectsCaseAliases(t *testing.T) {
	root := t.TempDir()
	err := run([]string{"--result-file", filepath.Join(root, "Report.json"), "--retrieve", "@com.example.app:documents/report.json=" + filepath.Join(root, "report.json")})
	if err == nil || !strings.Contains(err.Error(), "output path collision") {
		t.Fatalf("error=%v", err)
	}
}

func TestOutputCollisionFollowsExistingRecordingSymlinks(t *testing.T) {
	for _, useLink := range []bool{false, true} {
		t.Run(map[bool]string{false: "target", true: "link"}[useLink], func(t *testing.T) {
			root := t.TempDir()
			record := filepath.Join(root, "recordings")
			if err := os.Mkdir(record, 0700); err != nil {
				t.Fatal(err)
			}
			target := filepath.Join(root, "report.json")
			link := filepath.Join(record, "000000.jpg")
			if err := os.Symlink(target, link); err != nil {
				t.Fatal(err)
			}
			result := target
			if useLink {
				result = link
			}
			err := run([]string{"--record-dir", record, "--result-file", result})
			if err == nil || !strings.Contains(err.Error(), "output path collision") {
				t.Fatalf("error=%v", err)
			}
		})
	}
}
