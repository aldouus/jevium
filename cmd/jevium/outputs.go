package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/aldous/jevium/internal/appium"
)

var recordingName = regexp.MustCompile(`^[0-9]+\.jpg$`)

func canonicalOutputPath(path string) (string, error) {
	return resolveOutputPath(path, 0)
}

func resolveOutputPath(path string, depth int) (string, error) {
	if depth > 64 {
		return "", fmt.Errorf("output path exceeds symlink resolution limit")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err == nil {
		return resolved, nil
	}
	if !errors.Is(err, os.ErrNotExist) || filepath.Dir(absolute) == absolute {
		return "", err
	}
	if info, statErr := os.Lstat(absolute); statErr == nil && info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(absolute)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(absolute), target)
		}
		return resolveOutputPath(target, depth+1)
	}
	parent, err := resolveOutputPath(filepath.Dir(absolute), depth+1)
	if err != nil {
		return "", err
	}
	return filepath.Join(parent, filepath.Base(absolute)), nil
}

func validateOutputs(record, result, coverage string, retrievals []appium.Retrieval) error {
	var recording string
	var reservedPaths []string
	if record != "" {
		var err error
		recording, err = canonicalOutputPath(record)
		if err != nil {
			return err
		}
		entries, err := os.ReadDir(recording)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		for _, entry := range entries {
			if !reservedRecordingName(entry.Name()) {
				continue
			}
			path, err := canonicalOutputPath(filepath.Join(recording, entry.Name()))
			if err != nil {
				return err
			}
			reservedPaths = append(reservedPaths, path)
		}
	}
	paths := []string{result, coverage}
	for _, item := range retrievals {
		paths = append(paths, item.LocalPath)
	}
	seen := reservedPaths
	for _, path := range paths {
		if path == "" {
			continue
		}
		canonical, err := canonicalOutputPath(path)
		if err != nil {
			return err
		}
		parent, err := canonicalOutputPath(filepath.Dir(path))
		if err != nil {
			return err
		}
		reserved := recording != "" && (sameOutputPath(canonical, recording) || sameOutputPath(parent, recording) && reservedRecordingName(filepath.Base(path)) || sameOutputPath(filepath.Dir(canonical), recording) && reservedRecordingName(filepath.Base(canonical)))
		for _, other := range seen {
			reserved = reserved || sameOutputPath(canonical, other)
		}
		if reserved {
			return fmt.Errorf("output path collision: %s", path)
		}
		seen = append(seen, canonical)
	}
	return nil
}

func reservedRecordingName(name string) bool {
	name = strings.ToLower(name)
	return name == "actions.log" || recordingName.MatchString(name)
}

func sameOutputPath(a, b string) bool {
	if strings.EqualFold(a, b) {
		return true
	}
	aInfo, aErr := os.Stat(a)
	bInfo, bErr := os.Stat(b)
	return aErr == nil && bErr == nil && os.SameFile(aInfo, bInfo)
}
