package appium

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
)

const maxFixtureBytes = 20 << 20

type Fixture struct {
	LocalPath  string
	RemotePath string
}
type loadedFixture struct {
	Fixture
	data []byte
}

var fixtureDestination = regexp.MustCompile(`^@[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)+:documents/([^\x00-\x1f]+)$`)

func ParseFixture(value string) (Fixture, error) {
	local, remote, ok := strings.Cut(value, "=")
	if !ok || local == "" || remote == "" {
		return Fixture{}, fmt.Errorf("fixture must be LOCAL=@bundle.id:documents/filename")
	}
	return Fixture{local, remote}, nil
}

func loadFixtures(fixtures []Fixture) ([]loadedFixture, error) {
	var out []loadedFixture
	seen := map[string]bool{}
	for _, f := range fixtures {
		match := fixtureDestination.FindStringSubmatch(f.RemotePath)
		if match == nil || path.Clean(match[1]) != match[1] || strings.HasPrefix(match[1], "/") || strings.HasPrefix(match[1], "../") || match[1] == ".." || match[1] == "." || strings.Contains(match[1], "\\") {
			return nil, fmt.Errorf("fixture destination must name a file inside @bundle.id:documents/")
		}
		if seen[f.RemotePath] {
			return nil, fmt.Errorf("duplicate fixture destination %q", f.RemotePath)
		}
		seen[f.RemotePath] = true
		file, err := os.Open(f.LocalPath)
		if err != nil {
			return nil, fmt.Errorf("open fixture: %w", err)
		}
		stat, err := file.Stat()
		if err != nil {
			file.Close()
			return nil, err
		}
		if !stat.Mode().IsRegular() || stat.Size() > maxFixtureBytes {
			file.Close()
			return nil, fmt.Errorf("fixture must be a regular file no larger than 20 MiB")
		}
		data, err := io.ReadAll(io.LimitReader(file, maxFixtureBytes+1))
		closeErr := file.Close()
		if err != nil {
			return nil, err
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(data) > maxFixtureBytes {
			return nil, fmt.Errorf("fixture exceeds 20 MiB")
		}
		out = append(out, loadedFixture{f, data})
	}
	return out, nil
}

func (d *Device) provision(fixtures []loadedFixture) error {
	for _, f := range fixtures {
		if err := d.call(http.MethodPost, d.path("/appium/device/push_file"), map[string]string{"path": f.RemotePath, "data": base64.StdEncoding.EncodeToString(f.data)}, nil); err != nil {
			return fmt.Errorf("fixture push failed (not retried): %w", err)
		}
		var encoded string
		if err := d.call(http.MethodPost, d.path("/appium/device/pull_file"), map[string]string{"path": f.RemotePath}, &encoded); err != nil {
			return fmt.Errorf("fixture pushed but verification failed: %w", err)
		}
		received, err := base64.StdEncoding.DecodeString(encoded)
		if err != nil || !bytes.Equal(received, f.data) {
			return fmt.Errorf("fixture pushed but device bytes did not match local fixture")
		}
	}
	return nil
}
