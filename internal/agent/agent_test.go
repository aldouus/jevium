package agent_test

import (
	"errors"
	"testing"

	"github.com/aldous/jevium/internal/agent"
	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
	"github.com/aldous/jevium/internal/stale"
)

type fakeSurface struct {
	page     page.Page
	fresh    bool
	acted    []page.Action
	observe  int
	freshN   int
	fillMove bool
}

func (f *fakeSurface) Observe(bool) (page.Page, error) {
	f.observe++
	return f.page, nil
}
func (f *fakeSurface) Fresh(_ page.Page, action *page.Action) bool {
	f.freshN++
	if f.fillMove && action != nil && action.Kind == "fill" && f.freshN > 2 {
		return false
	}
	return f.fresh
}
func (f *fakeSurface) Act(a page.Action, _ page.Page, _ *string) error {
	f.acted = append(f.acted, a)
	return nil
}
func (f *fakeSurface) Close() error { return nil }

type fakeChooser struct {
	d   policy.Decision
	err error
}

func (f fakeChooser) Choose(page.Page, string, []policy.History) (policy.Decision, error) {
	return f.d, f.err
}
func (f fakeChooser) FieldText(policy.FieldInput) (string, policy.TextHelper, error) {
	return "example.com", policy.TextHelper{Model: "test", LatencyMS: 1}, nil
}

func sample() page.Page {
	return page.WithFingerprint(page.Page{
		URL: "app://x", Title: "x", Text: "SpringBoard",
		Actions: []page.Action{
			{ID: "e3", Kind: "click", Label: "Safari", Node: "n-safari"},
			{ID: "e1", Kind: "fill", Label: "Address", Node: "n-address", Rect: &page.Rect{X: 48, Y: 54, W: 240, H: 32}},
			{ID: "wait", Kind: "wait", Label: "Wait"},
		},
	})
}

func TestStaleDecisionDoesNotMutate(t *testing.T) {
	p := sample()
	s := &fakeSurface{page: p, fresh: false}
	a, err := agent.New(s, fakeChooser{d: policy.Decision{Choice: "e3", Operation: "CLICK", Probabilities: map[string]float64{"e3": 1}}}, "Open Safari", false, "")
	if err != nil {
		t.Fatal(err)
	}
	a.State.Decision = &policy.Decision{Choice: "e3", Operation: "CLICK", Probabilities: map[string]float64{"e3": 1}}
	a.State.Status = "predicted"
	err = a.Command("act", p.Fingerprint)
	if err == nil {
		t.Fatal("expected stale")
	}
	if !agent.IsStale(err) {
		t.Fatalf("want typed stale, got %T %v", err, err)
	}
	if !errors.Is(err, stale.Error{}) {
		t.Fatalf("errors.Is missed %v", err)
	}
	if len(s.acted) != 0 {
		t.Fatalf("acted=%v", s.acted)
	}
	if a.State.Decision != nil {
		t.Fatal("decision not consumed")
	}
}

func TestIsStaleDoesNotGrepStrings(t *testing.T) {
	if agent.IsStale(errors.New("page changed since the decision. Observe again")) {
		t.Fatal("string-shaped errors must not count as stale")
	}
	if !agent.IsStale(stale.Error{Msg: "anything"}) {
		t.Fatal("typed stale missed")
	}
}

func TestTickRecoversWhenActIsStale(t *testing.T) {
	p := sample()
	s := &fakeSurface{page: p, fresh: false}
	a, err := agent.New(s, fakeChooser{d: policy.Decision{Choice: "e3", Operation: "CLICK", Probabilities: map[string]float64{"e3": 1}}}, "Open Safari", false, "")
	if err != nil {
		t.Fatal(err)
	}
	n := s.observe
	if err := a.Command("tick", ""); err != nil {
		t.Fatal(err)
	}
	if a.State.Status != "ready" || a.State.Decision != nil {
		t.Fatalf("state=%s decision=%v", a.State.Status, a.State.Decision)
	}
	if len(s.acted) != 0 {
		t.Fatal("acted on stale page")
	}
	if s.observe <= n {
		t.Fatal("did not reobserve")
	}
}

func TestTickReobservesWithoutActionOnStale(t *testing.T) {
	p := sample()
	s := &fakeSurface{page: p, fresh: false}
	a, err := agent.New(s, fakeChooser{err: stale.Error{Msg: "page changed since the decision. Observe again"}}, "Open Safari", false, "")
	if err != nil {
		t.Fatal(err)
	}
	n := s.observe
	if err := a.Command("tick", ""); err != nil {
		t.Fatal(err)
	}
	if a.State.Status != "ready" || a.State.Decision != nil {
		t.Fatalf("state=%s", a.State.Status)
	}
	if s.observe <= n {
		t.Fatal("did not reobserve")
	}
	if len(s.acted) != 0 {
		t.Fatal("acted")
	}
}

func TestDoneIsNotSuccessWithoutVisibleEvidence(t *testing.T) {
	p := sample()
	s := &fakeSurface{page: p, fresh: true}
	a, err := agent.New(s, fakeChooser{d: policy.Decision{Choice: "DONE", Operation: "DONE"}}, "Stop when Example Domain is visible.", false, "")
	if err != nil {
		t.Fatal(err)
	}
	a.State.Decision = &policy.Decision{Choice: "DONE", Operation: "DONE"}
	a.State.Status = "predicted"
	if err := a.Command("act", p.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if a.State.Status == "done" {
		t.Fatal("DONE without evidence reported success")
	}
	if a.State.Status != "ready" {
		t.Fatalf("status=%s want ready to keep going", a.State.Status)
	}
}

func TestDoneSucceedsWhenGoalIsVisible(t *testing.T) {
	p := page.WithFingerprint(page.Page{URL: "https://example.com", Title: "Example Domain", Text: "Example Domain"})
	s := &fakeSurface{page: p, fresh: true}
	a, err := agent.New(s, fakeChooser{d: policy.Decision{Choice: "DONE", Operation: "DONE"}}, "Stop when Example Domain is visible.", false, "")
	if err != nil {
		t.Fatal(err)
	}
	a.State.Decision = &policy.Decision{Choice: "DONE", Operation: "DONE"}
	a.State.Status = "predicted"
	if err := a.Command("act", p.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if a.State.Status != "done" {
		t.Fatalf("status=%s", a.State.Status)
	}
}

func TestFailedDoneBlocksAfterRetries(t *testing.T) {
	p := sample()
	s := &fakeSurface{page: p, fresh: true}
	a, err := agent.New(s, fakeChooser{d: policy.Decision{Choice: "DONE", Operation: "DONE"}}, "Stop when Example Domain is visible.", false, "")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		a.State.Decision = &policy.Decision{Choice: "DONE", Operation: "DONE"}
		a.State.Status = "predicted"
		if err := a.Command("act", p.Fingerprint); err != nil {
			t.Fatal(err)
		}
	}
	if a.State.Status != "blocked" {
		t.Fatalf("status=%s want blocked after failed DONE checks", a.State.Status)
	}
}

func TestInjectedVerifierCanAcceptOrReject(t *testing.T) {
	p := sample()
	s := &fakeSurface{page: p, fresh: true}
	a, err := agent.New(s, fakeChooser{}, "anything", false, "")
	if err != nil {
		t.Fatal(err)
	}
	a.VerifyDone = func(string, page.Page) bool { return true }
	a.State.Decision = &policy.Decision{Choice: "DONE", Operation: "DONE"}
	a.State.Status = "predicted"
	if err := a.Command("act", p.Fingerprint); err != nil {
		t.Fatal(err)
	}
	if a.State.Status != "done" {
		t.Fatalf("injected verifier ignored: %s", a.State.Status)
	}
}

func TestFillRefreshesBeforeTyping(t *testing.T) {
	p := sample()
	s := &fakeSurface{page: p, fresh: true, fillMove: true}
	a, err := agent.New(s, fakeChooser{d: policy.Decision{Choice: "e1", Operation: "TYPE_TEXT"}}, "type example.com", false, "")
	if err != nil {
		t.Fatal(err)
	}
	a.State.Decision = &policy.Decision{Choice: "e1", Operation: "TYPE_TEXT", Probabilities: map[string]float64{"e1": 1}}
	a.State.Status = "predicted"
	err = a.Command("act", p.Fingerprint)
	if err == nil {
		t.Fatal("expected stale before typing a moved field")
	}
	if !agent.IsStale(err) {
		t.Fatalf("want typed stale got %T %v", err, err)
	}
	if len(s.acted) != 0 {
		t.Fatalf("typed into moved field: %v", s.acted)
	}
}
