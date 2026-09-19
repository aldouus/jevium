package env

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func dotenvValue(raw string) string {
	value := strings.TrimSpace(raw)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return value
}

func CandidateEnvFiles(explicit string) []string {
	if explicit != "" {
		return []string{explicit}
	}
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	return []string{
		filepath.Join(cwd, ".env"),
		filepath.Join(moduleRoot(), ".env"),
	}
}

func moduleRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "."
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func Load(path string) error {
	seen := map[string]struct{}{}
	for _, envPath := range CandidateEnvFiles(path) {
		resolved, err := filepath.Abs(envPath)
		if err != nil {
			continue
		}
		if _, ok := seen[resolved]; ok {
			continue
		}
		if _, err := os.Stat(envPath); err != nil {
			continue
		}
		seen[resolved] = struct{}{}
		if err := applyDotenv(envPath); err != nil {
			return err
		}
	}
	return nil
}

func applyDotenv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || !strings.Contains(line, "=") {
			continue
		}
		key, raw, _ := strings.Cut(line, "=")
		value := dotenvValue(raw)
		if value == "" {
			continue
		}
		key = strings.TrimSpace(key)
		if os.Getenv(key) == "" {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return scanner.Err()
}

func Require(name, purpose string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value != "" {
		return value, nil
	}
	return "", fmt.Errorf("%s is required %s. Export it in the shell or set it in .env", name, purpose)
}

func Get(name, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v
	}
	return fallback
}

func Atoi(name string, fallback int) int {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
