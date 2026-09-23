package policy

import (
	"github.com/aldous/jevium/internal/page"
	"strings"
)

func ScopedActions(p page.Page, scope string) []page.Action {
	out := make([]page.Action, 0, len(p.Actions))
	for _, a := range p.Actions {
		if a.Scope == "" {
			a.Scope = "native"
			if strings.HasPrefix(p.URL, "http://") || strings.HasPrefix(p.URL, "https://") {
				a.Scope = "web"
			}
		}
		if scope == "" || scope == "all" || scope == a.Scope || a.Kind == "wait" {
			out = append(out, a)
		}
	}
	return out
}
