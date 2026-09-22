package appium

import (
	"encoding/json"
	"fmt"
)

type sessionCaps struct {
	PlatformName       string `json:"platformName"`
	AutomationName     string `json:"appium:automationName"`
	UDID               string `json:"appium:udid"`
	XcodeOrgID         string `json:"appium:xcodeOrgId"`
	XcodeSigningID     string `json:"appium:xcodeSigningId"`
	UpdatedWDABundleID string `json:"appium:updatedWDABundleId"`
	WDALocalPort       int    `json:"appium:wdaLocalPort"`
	MJPEGServerPort    int    `json:"appium:mjpegServerPort"`
	WDALaunchTimeout   int    `json:"appium:wdaLaunchTimeout"`
	NewCommandTimeout  int    `json:"appium:newCommandTimeout"`
	NoReset            bool   `json:"appium:noReset"`
	BundleID           string `json:"appium:bundleId,omitempty"`
	DerivedDataPath    string `json:"appium:derivedDataPath,omitempty"`
}

type newSessionRequest struct {
	Capabilities struct {
		AlwaysMatch sessionCaps `json:"alwaysMatch"`
	} `json:"capabilities"`
}

type sessionValue struct {
	SessionID    string          `json:"sessionId"`
	Capabilities json.RawMessage `json:"capabilities"`
}

type executeRequest[T any] struct {
	Script string `json:"script"`
	Args   []T    `json:"args"`
}

type point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type bundleArg struct {
	BundleID string `json:"bundleId"`
}

type dragArg struct {
	FromX    float64 `json:"fromX"`
	FromY    float64 `json:"fromY"`
	ToX      float64 `json:"toX"`
	ToY      float64 `json:"toY"`
	Duration float64 `json:"duration"`
}

type w3cActionsRequest struct {
	Actions []w3cSource `json:"actions"`
}

type w3cSource struct {
	Type    string    `json:"type"`
	ID      string    `json:"id"`
	Actions []w3cTick `json:"actions"`
}

type w3cTick struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type windowRect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

func decodeWire(status int, method, path string, data []byte, dest any) error {
	unexpected := DeviceError{Msg: fmt.Sprintf("Appium returned unexpected JSON (%d) for %s %s", status, method, path)}
	if !json.Valid(data) {
		return DeviceError{Msg: fmt.Sprintf("Appium returned non-JSON (%d) for %s %s", status, method, path)}
	}
	var env struct {
		Value json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(data, &env); err != nil || env.Value == nil {
		return unexpected
	}
	if err := wireError(env.Value); err != nil {
		return err
	}
	if status >= 400 {
		return DeviceError{Msg: fmt.Sprintf("Appium returned HTTP %d for %s %s", status, method, path)}
	}
	if dest == nil || string(env.Value) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Value, dest); err != nil {
		return unexpected
	}
	return nil
}

func wireError(value json.RawMessage) error {
	if len(value) == 0 || value[0] != '{' {
		return nil
	}
	var probe struct {
		Error   json.RawMessage `json:"error"`
		Message *string         `json:"message"`
	}
	if err := json.Unmarshal(value, &probe); err != nil {
		return DeviceError{Msg: "Appium returned unexpected JSON"}
	}
	if len(probe.Error) == 0 || string(probe.Error) == "null" {
		return nil
	}
	var name string
	if err := json.Unmarshal(probe.Error, &name); err != nil {
		return DeviceError{Msg: "Appium returned unexpected JSON: value.error"}
	}
	if name == "" {
		return nil
	}
	msg := name
	if probe.Message != nil && *probe.Message != "" {
		msg = *probe.Message
	}
	return DeviceError{Msg: msg}
}
