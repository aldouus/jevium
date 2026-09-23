package appium

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Retrieval struct {
	RemotePath string
	LocalPath  string
}

func ParseRetrieval(value string) (Retrieval, error) {
	remote, local, ok := strings.Cut(value, "=")
	if !ok || remote == "" || local == "" {
		return Retrieval{}, fmt.Errorf("retrieval must be @bundle.id:documents/filename=LOCAL")
	}
	return Retrieval{remote, local}, nil
}

func validateRetrievals(items []Retrieval) error {
	seen := map[string]bool{}
	for _, item := range items {
		if !validDocumentPath(item.RemotePath) {
			return fmt.Errorf("retrieval source must name a file inside @bundle.id:documents/")
		}
		if item.LocalPath == "" {
			return fmt.Errorf("retrieval destination must name a new local file")
		}
		absolute, err := filepath.Abs(item.LocalPath)
		if err != nil {
			return err
		}
		parent, err := filepath.EvalSymlinks(filepath.Dir(absolute))
		if err != nil {
			return fmt.Errorf("retrieval destination parent: %w", err)
		}
		info, err := os.Stat(parent)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("retrieval destination parent is not a directory")
		}
		key := filepath.Join(parent, filepath.Base(absolute))
		if seen[key] {
			return fmt.Errorf("duplicate retrieval destination %q", item.LocalPath)
		}
		seen[key] = true
		if _, err := os.Lstat(absolute); err == nil {
			return fmt.Errorf("retrieval destination already exists: %s", item.LocalPath)
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return nil
}

func (d *Device) RetrieveArtifacts() error {
	if err := validateRetrievals(d.cfg.Retrievals); err != nil {
		return err
	}
	for _, item := range d.cfg.Retrievals {
		var encoded *string
		limit := int64(base64.StdEncoding.EncodedLen(maxFixtureBytes) + 4096)
		if err := d.callLimited(http.MethodPost, d.path("/appium/device/pull_file"), map[string]string{"path": item.RemotePath}, &encoded, limit); err != nil {
			return fmt.Errorf("retrieve %s: %w", item.RemotePath, err)
		}
		if encoded == nil {
			return fmt.Errorf("artifact response contained no file data")
		}
		if len(*encoded) > base64.StdEncoding.EncodedLen(maxFixtureBytes) {
			return fmt.Errorf("artifact exceeds 20 MiB")
		}
		data, err := base64.StdEncoding.Strict().DecodeString(*encoded)
		if err != nil {
			return fmt.Errorf("artifact response is not valid base64: %w", err)
		}
		if len(data) > maxFixtureBytes {
			return fmt.Errorf("artifact exceeds 20 MiB")
		}
		if err := saveArtifact(item.LocalPath, data); err != nil {
			return err
		}
	}
	return nil
}

func saveArtifact(destination string, data []byte) (err error) {
	file, err := os.CreateTemp(filepath.Dir(destination), ".jevium-artifact-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer func() { err = errors.Join(err, os.Remove(temporary)) }()
	if _, err = file.Write(data); err != nil {
		return errors.Join(err, file.Close())
	}
	if err = file.Sync(); err != nil {
		return errors.Join(err, file.Close())
	}
	if err = file.Close(); err != nil {
		return err
	}
	if err = os.Link(temporary, destination); err != nil {
		return fmt.Errorf("publish artifact without overwriting %s: %w", destination, err)
	}
	return nil
}
