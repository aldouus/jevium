package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/appium"
	"github.com/aldous/jevium/internal/chrome"
	"github.com/aldous/jevium/internal/env"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
	"github.com/aldous/jevium/internal/rlimit"
	"github.com/aldous/jevium/internal/tui"
	"github.com/aldous/jevium/internal/visual"
)

func main() {
	rlimit.Raise()
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) (runErr error) {
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
	visualOCR := fs.Bool("visual-ocr", false, "enable local macOS Vision text targets (screenshots stay local)")
	interactive := fs.Bool("tui", false, "Bubble Tea inspector")
	scope := fs.String("scope", "all", "allowed controls: all, web, or native")
	coveragePath := fs.String("coverage", "", "write JSON coverage report (multi-device inserts .UDID before file extension)")
	var expectedURLs goalList
	fs.Var(&expectedURLs, "audit-url", "expected page URL, repeatable; unvisited entries remain explicit")
	devices := fs.String("devices", "", "comma-separated UDIDs to run concurrently (requires --record-dir)")
	discover := fs.Bool("list-devices", false, "list connected devices using xcrun devicectl, without running a goal")
	mjpeg := fs.Int("mjpeg-server-port", env.Atoi("APPIUM_MJPEG_SERVER_PORT", 9101), "MJPEG port (first port for multiple devices)")
	derived := fs.String("derived-data-path", os.Getenv("APPIUM_DERIVED_DATA_PATH"), "Xcode derived-data directory (parent for multiple devices)")
	resultFile := fs.String("result-file", "", "write final run state as JSON")
	var goals goalList
	var fixtures goalList
	var retrievals goalList
	fs.Var(&retrievals, "retrieve", "retrieve @bundle.id:documents/filename=LOCAL after the run (repeatable; never overwrites local files)")
	fs.Var(&fixtures, "fixture", "provision LOCAL=@bundle.id:documents/filename before the goal (repeatable; overwrites destination)")
	var expectations goalList
	fs.Var(&expectations, "expect", "completion condition JSON {field,label,value}; repeat for conjunction")
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
	if *mode != "appium" && len(fixtures) > 0 {
		return fmt.Errorf("--fixture requires appium mode")
	}
	if *mode != "appium" && len(retrievals) > 0 {
		return fmt.Errorf("--retrieve requires appium mode")
	}
	if *mode != "appium" && *visualOCR {
		return fmt.Errorf("--visual-ocr requires appium mode")
	}
	if len(expectedURLs) > 0 && *coveragePath == "" {
		return fmt.Errorf("--audit-url requires --coverage")
	}
	conditions, err := agent.ParseExpectations(expectations)
	if err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	if *scope != "all" && *scope != "web" && *scope != "native" {
		return fmt.Errorf("invalid scope %q: use all, web, or native", *scope)
	}
	if *discover {
		if *devices != "" || len(goals) != 0 {
			return fmt.Errorf("--list-devices cannot run goals or --devices")
		}
		return listDevices(os.Stdout)
	}
	deviceConfig := appium.Config{StartURL: *startURL, AllowedApps: allowedApps, BundleID: *bundle, DeviceControls: *deviceControls}
	if *visualOCR {
		deviceConfig.Visual = visual.MacOCR{}
	}
	for _, value := range retrievals {
		item, err := appium.ParseRetrieval(value)
		if err != nil {
			return err
		}
		deviceConfig.Retrievals = append(deviceConfig.Retrievals, item)
	}
	for _, value := range fixtures {
		item, err := appium.ParseFixture(value)
		if err != nil {
			return err
		}
		deviceConfig.Fixtures = append(deviceConfig.Fixtures, item)
	}
	if *devices != "" {
		explicitUDID := false
		fs.Visit(func(f *flag.Flag) {
			if f.Name == "udid" {
				explicitUDID = true
			}
		})
		if explicitUDID {
			return fmt.Errorf("choose --devices or --udid, not both")
		}
		if *mode != "appium" || *interactive || *session != "" || *resultFile != "" {
			return fmt.Errorf("--devices requires appium mode without --tui, --session-id, or --result-file")
		}
		if len(goals) == 0 {
			return fmt.Errorf("supply --goal")
		}
		jobs, err := deviceJobs(*devices, *record, *derived, *wda, *mjpeg)
		if err != nil {
			return err
		}
		childArgs := []string{"--scope", *scope}
		childArgs = append(childArgs, appiumChildArgs(deviceConfig)...)
		for _, expectation := range expectations {
			childArgs = append(childArgs, "--expect", expectation)
		}
		for _, url := range expectedURLs {
			childArgs = append(childArgs, "--audit-url", url)
		}
		if *coveragePath != "" {
			for i := range jobs {
				jobs[i].Coverage = deviceOutputPath(*coveragePath, jobs[i].UDID)
			}
		}
		for i := range jobs {
			for _, item := range deviceConfig.Retrievals {
				jobs[i].Retrievals = append(jobs[i].Retrievals, appium.Retrieval{RemotePath: item.RemotePath, LocalPath: deviceOutputPath(item.LocalPath, jobs[i].UDID)})
			}
			cfg := deviceConfig
			cfg.Retrievals = jobs[i].Retrievals
			if err := appium.ValidateConfig(cfg); err != nil {
				return err
			}
		}
		if _, err := env.Require("TYPESAFE_API_KEY", "to call TypeSafe Jev"); err != nil {
			return err
		}
		executable, err := os.Executable()
		if err != nil {
			return err
		}
		return runDevices(jobs, executable, *appiumURL, *bundle, goals, os.Stdout, runDeviceProcess, childArgs...)
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
	chooser := policy.Client{Scope: *scope}
	var surface agent.Surface
	var device *appium.Device
	switch *mode {
	case "appium":
		deviceConfig.URL, deviceConfig.UDID, deviceConfig.SessionID = *appiumURL, *udid, *session
		deviceConfig.WDALocalPort, deviceConfig.MJPEGServerPort, deviceConfig.DerivedDataPath = *wda, *mjpeg, *derived
		device, err = appium.New(deviceConfig)
		surface = device
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
	if *coveragePath != "" {
		if err := a.EnableCoverage(*coveragePath, expectedURLs); err != nil {
			return err
		}
	}
	if len(conditions) > 0 {
		a.VerifyDone = func(_ string, p page.Page) bool { return agent.ExpectationsMatch(conditions, p) }
	}
	if *resultFile != "" {
		defer func() {
			data, err := json.Marshal(a.State)
			if err == nil {
				err = os.WriteFile(*resultFile, data, 0o600)
			}
			runErr = errors.Join(runErr, err)
		}()
	}
	if *interactive {
		err = tui.Run(a)
	} else {
		err = a.Run()
	}
	if device != nil {
		err = errors.Join(err, device.RetrieveArtifacts())
	}
	if err != nil {
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
