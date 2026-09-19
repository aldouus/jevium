package env_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aldous/jevium/internal/env"
)

func TestLoadKeepsExportedKeyOverBlankDotenv(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "from-shell")
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("TYPESAFE_API_KEY=\nTYPESAFE_MODEL=jev-latest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TYPESAFE_MODEL", "")
	os.Unsetenv("TYPESAFE_MODEL")
	if err := env.Load(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("TYPESAFE_API_KEY") != "from-shell" {
		t.Fatalf("key=%s", os.Getenv("TYPESAFE_API_KEY"))
	}
	if os.Getenv("TYPESAFE_MODEL") != "jev-latest" {
		t.Fatalf("model=%s", os.Getenv("TYPESAFE_MODEL"))
	}
}

func TestRequireFailsClosed(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	if _, err := env.Require("TYPESAFE_API_KEY", "to call TypeSafe Jev"); err == nil {
		t.Fatal("expected error")
	}
}

func TestRequireFailsOnWhitespace(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "   ")
	if _, err := env.Require("TYPESAFE_API_KEY", "to call TypeSafe Jev"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAtoiUsesEnvThenFallback(t *testing.T) {
	t.Setenv("APPIUM_WDA_LOCAL_PORT", "")
	if env.Atoi("APPIUM_WDA_LOCAL_PORT", 8101) != 8101 {
		t.Fatal("empty env should use fallback")
	}
	t.Setenv("APPIUM_WDA_LOCAL_PORT", "not-a-port")
	if env.Atoi("APPIUM_WDA_LOCAL_PORT", 8101) != 8101 {
		t.Fatal("invalid env should use fallback")
	}
	t.Setenv("APPIUM_WDA_LOCAL_PORT", "9100")
	if env.Atoi("APPIUM_WDA_LOCAL_PORT", 8101) != 9100 {
		t.Fatal("valid env ignored")
	}
}

func TestLoadSkipsBlankDotenvValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("EMPTY_ONLY=\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("EMPTY_ONLY", "keep-me")
	if err := env.Load(path); err != nil {
		t.Fatal(err)
	}
	if os.Getenv("EMPTY_ONLY") != "keep-me" {
		t.Fatalf("EMPTY_ONLY=%q", os.Getenv("EMPTY_ONLY"))
	}
}
