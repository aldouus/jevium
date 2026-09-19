package main

import (
	"strings"
	"testing"
)

func TestRunAppiumMissingUDID(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "test")
	t.Setenv("APPIUM_UDID", "")
	err := run([]string{"--mode", "appium", "--udid", "", "--goal", "open safari"})
	if err == nil {
		t.Fatal("expected missing UDID to fail before Appium starts")
	}
	if !strings.Contains(strings.ToLower(err.Error()), "udid") {
		t.Fatalf("err=%v", err)
	}
}
