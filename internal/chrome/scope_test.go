package chrome_test

import (
	"github.com/aldous/jevium/internal/chrome"
	"github.com/aldous/jevium/internal/policy"
	"testing"
)

func TestChromeDOMScopeIndependentOfURLScheme(t *testing.T) {
	for _, url := range []string{"file:///tmp/test.html", "about:blank", "data:text/html,hello"} {
		raw := snapshotRaw(7)
		raw["url"] = url
		p, err := chrome.FromRuntime(&fakeRuntime{eval: func(string) (any, error) { return raw, nil }}).Observe(false)
		if err != nil {
			t.Fatal(err)
		}
		got := policy.ScopedActions(p, "web")
		if len(got) != 3 || got[0].Scope != "web" {
			t.Fatalf("%s controls=%+v", url, got)
		}
	}
}
