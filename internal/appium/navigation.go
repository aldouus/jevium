package appium

import (
	"fmt"
	"github.com/aldous/jevium/internal/page"
	"net/http"
	"net/url"
	"regexp"
)

var bundlePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+(?:\.[A-Za-z0-9_-]+)+$`)

func validateNavigation(cfg Config) error {
	if cfg.StartURL != "" {
		u, err := url.Parse(cfg.StartURL)
		if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Hostname() == "" || u.User != nil {
			return fmt.Errorf("--url must be an absolute HTTP(S) URL without credentials")
		}
		if cfg.BundleID == "" {
			return fmt.Errorf("--url requires a bundle id")
		}
		if cfg.BundleID == "com.apple.springboard" {
			return fmt.Errorf("--url requires a browser bundle id, not SpringBoard")
		}
	}
	seen := map[string]bool{}
	for _, app := range cfg.AllowedApps {
		if !bundlePattern.MatchString(app) || seen[app] {
			return fmt.Errorf("invalid or duplicate allowed app %q", app)
		}
		seen[app] = true
	}
	return nil
}

func (d *Device) openStartURL() error {
	if d.cfg.StartURL == "" {
		return nil
	}
	return d.execute("mobile: deepLink", []struct {
		URL      string `json:"url"`
		BundleID string `json:"bundleId"`
	}{{d.cfg.StartURL, d.cfg.BundleID}})
}

func (d *Device) allowedApp(bundle string) bool {
	for _, app := range d.cfg.AllowedApps {
		if app == bundle {
			return true
		}
	}
	return false
}

func (d *Device) navigationSnapshot(source string) (page.Page, error) {
	bundle := d.cfg.BundleID
	if len(d.cfg.AllowedApps) > 0 {
		var info struct {
			BundleID string `json:"bundleId"`
		}
		if err := d.call(http.MethodPost, d.path("/execute/sync"), executeRequest[struct{}]{Script: "mobile: activeAppInfo", Args: []struct{}{}}, &info); err != nil {
			return page.Page{}, err
		}
		if !bundlePattern.MatchString(info.BundleID) {
			return page.Page{}, DeviceError{Msg: "Appium activeAppInfo returned no valid bundle id"}
		}
		bundle = info.BundleID
	}
	p, err := d.secretSnapshot(source, bundle)
	if err != nil {
		return p, err
	}
	for i, app := range d.cfg.AllowedApps {
		p.Actions = append(p.Actions,
			page.Action{ID: fmt.Sprintf("activate_app_%d", i), Kind: "activate_app", Value: app, Label: "Launch or switch to configured app " + app},
			page.Action{ID: fmt.Sprintf("terminate_app_%d", i), Kind: "terminate_app", Value: app, Label: "Terminate configured app " + app + "; launch it in a separate decision to restart"})
	}
	return page.WithFingerprint(p), nil
}
