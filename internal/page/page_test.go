package page_test

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/aldous/jevium/internal/page"
)

func sample() page.Page {
	return page.Page{
		URL:   "https://example.com",
		Title: "Example",
		Text:  "Hello",
		Actions: []page.Action{
			{ID: "e1", Kind: "click", Label: "Go", Node: "n-1"},
		},
		Scroll:     page.Scroll{Height: 800},
		Screenshot: "one",
	}
}

func TestFingerprintIgnoresScreenshots(t *testing.T) {
	t.Parallel()
	a := sample()
	b := sample()
	b.Screenshot = "two"
	b.Title = "Other"
	if page.Fingerprint(a) != page.Fingerprint(b) {
		t.Fatal("screenshot or title affected fingerprint")
	}
}

func TestFingerprintChangesWhenNodeLabelOrKindChanges(t *testing.T) {
	t.Parallel()
	orig := page.Fingerprint(sample())
	cases := []struct {
		name   string
		mutate func(page.Page) page.Page
	}{
		{"node", func(p page.Page) page.Page {
			p.Actions = append([]page.Action(nil), p.Actions...)
			p.Actions[0].Node = "n-other"
			return p
		}},
		{"label", func(p page.Page) page.Page {
			p.Actions = append([]page.Action(nil), p.Actions...)
			p.Actions[0].Label = "Stop"
			return p
		}},
		{"kind", func(p page.Page) page.Page {
			p.Actions = append([]page.Action(nil), p.Actions...)
			p.Actions[0].Kind = "fill"
			return p
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if page.Fingerprint(c.mutate(sample())) == orig {
				t.Fatalf("%s change ignored", c.name)
			}
		})
	}
}

func TestFingerprintHasStableKeyOrder(t *testing.T) {
	t.Parallel()
	wantJSON := `{"url":"https://example.com","text":"Hello","actions":[{"id":"e1","kind":"click","label":"Go","role":"","value":"","current_value":"","node":"n-1","checked":"","expanded":"","selected":null}],"scroll":{"y":0,"height":800}}`
	sum := sha256.Sum256([]byte(wantJSON))
	want := fmt.Sprintf("%x", sum[:])
	if got := page.Fingerprint(sample()); got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestWithFingerprintSetsHash(t *testing.T) {
	t.Parallel()
	p := page.WithFingerprint(sample())
	if p.Fingerprint == "" || p.Fingerprint != page.Fingerprint(sample()) {
		t.Fatalf("fingerprint=%q", p.Fingerprint)
	}
}

func TestFindAction(t *testing.T) {
	t.Parallel()
	actions := sample().Actions
	got, ok := page.FindAction(actions, "e1")
	if !ok || got.Label != "Go" {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
	if _, ok := page.FindAction(actions, "missing"); ok {
		t.Fatal("found missing")
	}
}
