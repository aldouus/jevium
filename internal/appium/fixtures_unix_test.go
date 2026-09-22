//go:build darwin || linux

package appium

import (
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestFixtureRejectsFIFOWithoutWaitingForWriter(t *testing.T) {
	local := filepath.Join(t.TempDir(), "fifo")
	if err := syscall.Mkfifo(local, 0600); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { _, err := loadFixtures([]Fixture{{local, "@com.example.files:documents/fifo"}}); result <- err }()
	select {
	case err := <-result:
		if err == nil || err.Error() != "fixture must be a regular file no larger than 20 MiB" {
			t.Fatalf("error=%v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("fixture open waited for a FIFO writer")
	}
}
