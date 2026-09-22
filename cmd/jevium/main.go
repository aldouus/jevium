package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/chrome"
	"github.com/aldous/jevium/internal/env"
	"github.com/aldous/jevium/internal/policy"
	"github.com/aldous/jevium/internal/rlimit"
	"github.com/aldous/jevium/internal/tui"
)

func main() {
	rlimit.Raise()
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if err := env.Load(""); err != nil {
		return err
	}
	fs := flag.NewFlagSet("jevium", flag.ContinueOnError)
	mode := fs.String("mode", "appium", "appium (iPhone via XCUITest) or chrome (desktop browser via CDP)")
	udid := fs.String("udid", env.Get("APPIUM_UDID", ""), "device UDID")
	bundle := fs.String("bundle-id", env.Get("APPIUM_BUNDLE_ID", "com.apple.springboard"), "iOS bundle id")
	session := fs.String("session-id", os.Getenv("APPIUM_SESSION_ID"), "reuse an existing Appium session")
	appiumURL := fs.String("appium-url", env.Get("APPIUM_URL", "http://127.0.0.1:4723"), "Appium server URL")
	startURL := fs.String("url", "", "HTTP(S) page URL to open before the goal")
	var allowedApps goalList
	fs.Var(&allowedApps, "allow-app", "bundle id permitted for launch, switch, and terminate (repeatable)")
	wda := fs.Int("wda-local-port", env.Atoi("APPIUM_WDA_LOCAL_PORT", 8101), "WDA local port")
	deviceControls := fs.Bool("device-controls", false, "enable observed rotation, keyboard dismissal, lock and OS unlock controls")
	record := fs.String("record-dir", "", "optional screenshot directory")
	screenshots := fs.Bool("screenshots", false, "capture screenshots (not sent to Jev)")
	interactive := fs.Bool("tui", false, "Bubble Tea inspector")
	var goals goalList
	fs.Var(&goals, "goal", "natural-language goal (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *mode != "appium" && len(allowedApps) > 0 {
		return fmt.Errorf("--allow-app requires appium mode")
	}
	if *mode != "appium" && *deviceControls {
		return fmt.Errorf("--device-controls requires appium mode")
	}
	if _, err := env.Require("TYPESAFE_API_KEY", "to call TypeSafe Jev"); err != nil {
		return err
	}
	if len(goals) == 0 {
		return fmt.Errorf("supply --goal")
	}
	goal := strings.Join(goals, "\n")
	if *mode == "appium" && strings.TrimSpace(*udid) == "" {
		return fmt.Errorf("APPIUM_UDID is required for appium mode. Export it in the shell, set it in .env, or pass --udid")
	}
	chooser := policy.Client{}
	var surface agent.Surface
	var err error
	switch *mode {
	case "appium":
		surface, err = appium.New(appium.Config{
			StartURL: *startURL, AllowedApps: allowedApps,
			DeviceControls: *deviceControls,
			URL:            *appiumURL, UDID: *udid, BundleID: *bundle, SessionID: *session, WDALocalPort: *wda,
		})
	case "chrome":
		if *startURL == "" {
			return fmt.Errorf("chrome mode needs --url")
		}
		surface, err = chrome.New(*startURL)
	default:
		return fmt.Errorf("unknown mode %s", *mode)
	}
	if err != nil {
		return err
	}
	a, err := agent.New(surface, chooser, goal, *screenshots, *record)
	if err != nil {
		_ = surface.Close()
		return err
	}
	defer a.Close()
	if *interactive {
		return tui.Run(a)
	}
	if err := a.Run(); err != nil {
		return err
	}
	fmt.Printf("%5d ms  %d actions  %s\n", a.State.ElapsedMS, len(a.State.History), a.State.Status)
	fmt.Println(a.State.Page.URL)
	fmt.Println(a.State.Status)
	return nil
}

type goalList []string

func (g *goalList) String() string { return strings.Join(*g, " | ") }
func (g *goalList) Set(v string) error {
	*g = append(*g, v)
	return nil
}
