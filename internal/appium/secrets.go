package appium

import (
	"github.com/aldous/jevium/internal/page"
	"os"
	"strings"
)

func (d *Device) secretSnapshot(source string) (page.Page, error) {
	redact := func(s string) string {
		for _, ref := range d.cfg.SecretFields {
			if secret := os.Getenv(ref); secret != "" {
				s = strings.ReplaceAll(s, secret, "[redacted]")
			}
		}
		return s
	}
	p, err := snapshotFromSource(source, d.cfg.BundleID, nil, true, redact)
	if err != nil {
		return p, err
	}
	if len(d.cfg.SecretFields) > 0 {
		p.Source = ""
	}
	actions := make([]page.Action, 0, len(p.Actions))
	counts := map[string]int{}
	for _, a := range p.Actions {
		if a.Kind == "secure_fill" {
			counts[a.Label]++
		}
	}
	for _, a := range p.Actions {
		if a.Kind == "secure_fill" {
			if counts[a.Label] != 1 {
				continue
			}
			if _, ok := d.cfg.SecretFields[a.Label]; !ok {
				continue
			}
		}
		a.Label = redact(a.Label)
		a.Value = redact(a.Value)
		a.CurrentValue = redact(a.CurrentValue)
		if node, ok := a.Node.(string); ok {
			a.Node = redact(node)
		}
		actions = append(actions, a)
	}
	p.Actions = actions
	p.Text = redact(p.Text)
	p.Title = redact(p.Title)
	p.URL = redact(p.URL)
	return page.WithFingerprint(p), nil
}
