package main

import (
	"strings"
	"testing"
)

func TestRejectAppiumOnlyOptionInChrome(t *testing.T) {
	err := run([]string{"--mode", "chrome", "--allow-app=com.example.app"})
	if err == nil || err.Error() != "--allow-app requires appium mode" {
		t.Fatalf("error=%v", err)
	}
}

func TestRejectDeviceControlsInChrome(t *testing.T) {
	err := run([]string{"--mode", "chrome", "--device-controls=true"})
	if err == nil || err.Error() != "--device-controls requires appium mode" {
		t.Fatalf("error=%v", err)
	}
}

func TestRejectFixtureInChrome(t *testing.T) {
	err := run([]string{"--mode", "chrome", "--fixture=sample.pdf=@com.example.files:documents/sample.pdf"})
	if err == nil || err.Error() != "--fixture requires appium mode" {
		t.Fatalf("error=%v", err)
	}
}

func TestRejectRetrievalInChrome(t *testing.T) {
	err := run([]string{"--mode", "chrome", "--retrieve=@com.example.files:documents/download.pdf=download.pdf"})
	if err == nil || err.Error() != "--retrieve requires appium mode" {
		t.Fatalf("error=%v", err)
	}
}

func TestRejectVisualOCRInChrome(t *testing.T) {
	err := run([]string{"--mode", "chrome", "--visual-ocr=true"})
	if err == nil || err.Error() != "--visual-ocr requires appium mode" {
		t.Fatalf("error=%v", err)
	}
}

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
