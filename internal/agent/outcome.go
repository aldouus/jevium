package agent

import (
	"fmt"
	"github.com/aldous/jevium/internal/page"
)

// ActionOutcome separates a changed observation from proof of the intended effect.
func ActionOutcome(action page.Action, text *string, before, after page.Page) string {
	matches := []page.Action{}
	for _, candidate := range after.Actions {
		if fmt.Sprint(candidate.Node) == fmt.Sprint(action.Node) && candidate.Kind == action.Kind && candidate.Label == action.Label {
			matches = append(matches, candidate)
		}
	}
	if len(matches) == 1 {
		next := matches[0]
		switch action.Kind {
		case "fill":
			if text != nil && next.Value == *text {
				return "verified"
			}
		case "select":
			if next.CurrentValue == action.Value {
				return "verified"
			}
		case "click":
			if action.Checked != "" && next.Checked != "" && next.Checked != action.Checked {
				return "verified"
			}
			if action.Expanded != "" && next.Expanded != "" && next.Expanded != action.Expanded {
				return "verified"
			}
		}
	}
	if before.Fingerprint != after.Fingerprint {
		return "observed-change"
	}
	return "unchanged"
}

// Three repetitions of the same observed transitions are not forward progress.
// Waits are exempt: background work may legitimately remain unchanged.
func RepeatedCycle(h []HistoryEntry) bool {
	for period := 1; period <= 4; period++ {
		if len(h) < 3*period {
			continue
		}
		tail := h[len(h)-3*period:]
		repeated := true
		for i, e := range tail {
			ref := tail[i%period]
			if e.Kind == "wait" || e.Before == "" || e.After == "" || e.Before != ref.Before || e.After != ref.After || e.Kind != ref.Kind || e.Action != ref.Action {
				repeated = false
				break
			}
		}
		if repeated {
			return true
		}
	}
	return false
}
