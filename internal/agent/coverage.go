package agent

import (
	"encoding/json"
	"fmt"
	"github.com/aldous/jevium/internal/page"
	"os"
	"path/filepath"
)

type ControlCoverage struct {
	Label           string `json:"label"`
	Kind            string `json:"kind"`
	Attempts        int    `json:"attempts"`
	ObservedResults int    `json:"observed_results"`
	Changed         int    `json:"observed_changes"`
	Verified        int    `json:"verified_effects"`
}
type PageCoverage struct {
	Visited        bool                        `json:"visited"`
	Observations   map[string]bool             `json:"observations"`
	Controls       map[string]*ControlCoverage `json:"controls"`
	OmittedActions int                         `json:"max_omitted_actions"`
}
type Coverage struct {
	Status            string                   `json:"status"`
	CompleteInventory bool                     `json:"complete_inventory"`
	Pages             map[string]*PageCoverage `json:"pages"`
}

func (c *Coverage) page(url string) *PageCoverage {
	if c.Pages == nil {
		c.Pages = map[string]*PageCoverage{}
	}
	if c.Pages[url] == nil {
		c.Pages[url] = &PageCoverage{Observations: map[string]bool{}, Controls: map[string]*ControlCoverage{}}
	}
	return c.Pages[url]
}
func controlKey(a page.Action) string { return fmt.Sprintf("%v|%s|%s", a.Node, a.Kind, a.Label) }
func (c *Coverage) Observe(p page.Page) {
	entry := c.page(p.URL)
	entry.Visited = true
	entry.Observations[p.Fingerprint] = true
	entry.OmittedActions = max(entry.OmittedActions, p.OmittedActions)
	for _, a := range p.Actions {
		key := controlKey(a)
		if entry.Controls[key] == nil {
			entry.Controls[key] = &ControlCoverage{Label: a.Label, Kind: a.Kind}
		}
	}
}
func (c *Coverage) Attempt(p page.Page, a page.Action) {
	c.Observe(p)
	c.page(p.URL).Controls[controlKey(a)].Attempts++
}
func (c *Coverage) Result(before page.Page, a page.Action, after page.Page, text *string) {
	entry := c.page(before.URL).Controls[controlKey(a)]
	entry.ObservedResults++
	if before.Fingerprint != after.Fingerprint {
		entry.Changed++
	}
	// A changed page alone never proves the intended effect.
	matches := 0
	verified := false
	for _, next := range after.Actions {
		if controlKey(next) != controlKey(a) {
			continue
		}
		matches++
		verified = a.Kind == "fill" && text != nil && next.Value == *text || a.Kind == "click" && a.Checked != "" && next.Checked != "" && a.Checked != next.Checked
	}
	if matches == 1 && verified {
		entry.Verified++
	}
	c.Observe(after)
}
func (a *Agent) EnableCoverage(path string, expectedURLs []string) error {
	a.Coverage = &Coverage{Status: a.State.Status, Pages: map[string]*PageCoverage{}}
	for _, url := range expectedURLs {
		a.Coverage.page(url)
	}
	a.Coverage.Observe(a.State.Page)
	a.CoveragePath = path
	return a.saveCoverage()
}
func (a *Agent) saveCoverage() error {
	data, err := json.MarshalIndent(a.Coverage, "", "  ")
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(a.CoveragePath), ".jevium-coverage-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Rename(name, a.CoveragePath)
}
