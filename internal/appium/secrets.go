package appium

import (
	"github.com/aldous/jevium/internal/page"
	"os"
	"sort"
	"strings"
)

func (d *Device) secretSnapshot(source, bundle string) (page.Page, error) {
	secrets := make([]string, 0, len(d.cfg.SecretFields))
	for _, ref := range d.cfg.SecretFields {
		if secret := os.Getenv(ref); secret != "" {
			secrets = append(secrets, secret)
		}
	}
	sort.Slice(secrets, func(i, j int) bool {
		if len(secrets[i]) == len(secrets[j]) {
			return secrets[i] < secrets[j]
		}
		return len(secrets[i]) > len(secrets[j])
	})
	redact := func(s string) string {
		for _, secret := range secrets {
			s = strings.ReplaceAll(s, secret, "[redacted]")
		}
		return s
	}
	p, err := snapshotFromSource(source, bundle, nil, true, redact)
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
	for i := range p.Controls {
		control := &p.Controls[i]
		control.Label = redact(control.Label)
		if control.Value != nil {
			value := redact(*control.Value)
			control.Value = &value
		}
		if node, ok := control.Node.(string); ok {
			control.Node = redact(node)
		}
		control.Expanded = redact(control.Expanded)
		if selected, ok := control.Selected.(string); ok {
			control.Selected = redact(selected)
		}
	}
	p.Text = redact(p.Text)
	p.Title = redact(p.Title)
	p.URL = redact(p.URL)
	return page.WithFingerprint(p), nil
}
