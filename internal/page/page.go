package page

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

const MaxElementActions = 250
const MaxText = 6000

type Rect struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

type Scroll struct {
	Y      float64 `json:"y"`
	Height float64 `json:"height"`
}

type Action struct {
	Scope        string  `json:"scope,omitempty"`
	Destination  *Rect   `json:"destination,omitempty"`
	ID           string  `json:"id"`
	Kind         string  `json:"kind"`
	Label        string  `json:"label"`
	Role         string  `json:"role,omitempty"`
	Value        string  `json:"value,omitempty"`
	CurrentValue string  `json:"current_value,omitempty"`
	Node         any     `json:"node,omitempty"` // Chrome JSON number vs Appium string id
	Checked      string  `json:"checked,omitempty"`
	Selected     any     `json:"selected,omitempty"`
	Expanded     string  `json:"expanded,omitempty"`
	Rect         *Rect   `json:"rect,omitempty"`
	Delta        float64 `json:"delta,omitempty"`
	Direction    string  `json:"direction,omitempty"`
}

type Page struct {
	URL            string         `json:"url"`
	Title          string         `json:"title"`
	Text           string         `json:"text"`
	W              float64        `json:"w,omitempty"`
	H              float64        `json:"h,omitempty"`
	Scroll         Scroll         `json:"scroll"`
	Actions        []Action       `json:"actions"`
	OmittedActions int            `json:"omitted_actions,omitempty"`
	BundleID       string         `json:"bundle_id,omitempty"`
	Source         string         `json:"source,omitempty"`
	Fingerprint    string         `json:"fingerprint,omitempty"`
	Screenshot     string         `json:"screenshot,omitempty"`
	Marker         any            `json:"marker,omitempty"`
	PageKey        any            `json:"page_key,omitempty"`
	Guards         map[string]any `json:"guards,omitempty"`
}

func Fingerprint(p Page) string {
	type slim struct {
		Scope        string  `json:"scope,omitempty"`
		Destination  *Rect   `json:"destination,omitempty"`
		Delta        float64 `json:"delta,omitempty"`
		ID           string  `json:"id"`
		Kind         string  `json:"kind"`
		Label        string  `json:"label"`
		Role         string  `json:"role"`
		Value        string  `json:"value"`
		CurrentValue string  `json:"current_value"`
		Node         any     `json:"node"`
		Checked      string  `json:"checked"`
		Expanded     string  `json:"expanded"`
		Selected     any     `json:"selected"`
		Rect         *Rect   `json:"rect,omitempty"`
		Direction    string  `json:"direction,omitempty"`
	}
	type payload struct {
		URL     string `json:"url"`
		Text    string `json:"text"`
		Actions []slim `json:"actions"`
		Scroll  Scroll `json:"scroll"`
	}
	actions := make([]slim, 0, len(p.Actions))
	for _, a := range p.Actions {
		actions = append(actions, slim{
			Scope:       a.Scope,
			Destination: a.Destination, Delta: a.Delta,
			ID: a.ID, Kind: a.Kind, Label: a.Label, Role: a.Role, Value: a.Value,
			CurrentValue: a.CurrentValue, Node: a.Node, Checked: a.Checked,
			Expanded: a.Expanded, Selected: a.Selected,
			Rect: a.Rect, Direction: a.Direction,
		})
	}
	raw, err := json.Marshal(payload{
		URL: p.URL, Text: p.Text, Actions: actions, Scroll: p.Scroll,
	})
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return fmt.Sprintf("%x", sum[:])
}

func WithFingerprint(p Page) Page {
	p.Fingerprint = Fingerprint(p)
	return p
}

func FindAction(actions []Action, id string) (Action, bool) {
	for _, a := range actions {
		if a.ID == id {
			return a, true
		}
	}
	return Action{}, false
}
