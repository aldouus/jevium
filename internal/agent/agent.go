package agent

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
	"github.com/aldous/jevium/internal/stale"
)

func IsStale(err error) bool {
	return stale.Is(err)
}

type Surface interface {
	Observe(screenshot bool) (page.Page, error)
	Fresh(p page.Page, action *page.Action) bool
	Act(action page.Action, p page.Page, text *string) error
	Close() error
}

type Chooser interface {
	Choose(state page.Page, goal string, history []policy.History) (policy.Decision, error)
	FieldText(ctx policy.FieldInput) (string, policy.TextHelper, error)
}

type DoneCheck func(goal string, p page.Page) bool

type HistoryEntry struct {
	Outcome       string         `json:"outcome"`
	Before        string         `json:"before_fingerprint"`
	After         string         `json:"after_fingerprint,omitempty"`
	Step          int            `json:"step"`
	Action        string         `json:"action"`
	Kind          string         `json:"kind"`
	Choice        string         `json:"choice"`
	Probability   float64        `json:"probability"`
	Confidence    float64        `json:"confidence"`
	LatencyMS     int            `json:"latency_ms"`
	Text          string         `json:"text,omitempty"`
	TextHelper    string         `json:"text_helper,omitempty"`
	TextLatencyMS int            `json:"text_latency_ms,omitempty"`
	Operation     string         `json:"operation"`
	Target        string         `json:"target,omitempty"`
	PageChanged   *bool          `json:"page_changed"`
	URL           string         `json:"url"`
	Usage         map[string]any `json:"usage"`
	ExecutedMS    int            `json:"executed_ms"`
	ElapsedMS     int            `json:"elapsed_ms"`
}

func (e HistoryEntry) Summary() policy.History {
	return policy.History{Action: e.Action, Kind: e.Kind, Text: e.Text, PageChanged: e.PageChanged, Outcome: e.Outcome}
}

type State struct {
	Goal      string            `json:"goal"`
	Page      page.Page         `json:"page"`
	Decision  *policy.Decision  `json:"decision"`
	History   []HistoryEntry    `json:"history"`
	Status    string            `json:"status"`
	Decisions []policy.Decision `json:"decisions"`
	TextCalls []policy.TextCall `json:"text_calls"`
	ElapsedMS int               `json:"elapsed_ms"`
	StartedAt time.Time         `json:"-"`
	Record    bool              `json:"record"`
}

type Agent struct {
	Coverage     *Coverage
	CoveragePath string
	Surface      Surface
	Chooser      Chooser
	Screenshots  bool
	RecordDir    string
	VerifyDone   DoneCheck
	State        State
	pending      *pendingText
	failedDone   int
}

type pendingText struct {
	Context policy.FieldInput
	Text    string
	Helper  policy.TextHelper
}

func New(surface Surface, chooser Chooser, goal string, screenshots bool, recordDir string) (*Agent, error) {
	if goal == "" {
		return nil, fmt.Errorf("supply a task")
	}
	a := &Agent{Surface: surface, Chooser: chooser, Screenshots: screenshots || recordDir != "", RecordDir: recordDir, VerifyDone: GoalVisible}
	p, err := surface.Observe(a.Screenshots)
	if err != nil {
		_ = surface.Close()
		return nil, err
	}
	a.State = State{Goal: goal, Page: p, History: []HistoryEntry{}, Status: "ready", Decisions: []policy.Decision{}, TextCalls: []policy.TextCall{}}
	if recordDir != "" {
		if err := os.MkdirAll(recordDir, 0o755); err != nil {
			_ = surface.Close()
			return nil, err
		}
		if p.Screenshot != "" {
			b, err := base64.StdEncoding.DecodeString(p.Screenshot)
			if err == nil {
				_ = os.WriteFile(filepath.Join(recordDir, "000000.jpg"), b, 0o644)
			}
		}
		a.State.Record = true
	}
	return a, nil
}

func GoalVisible(goal string, p page.Page) bool {
	hay := strings.ToLower(strings.TrimSpace(p.Text) + "\n" + strings.TrimSpace(p.Title))
	hay = strings.TrimSpace(hay)
	if hay == "" {
		return false
	}
	needle := stopWhenNeedle(goal)
	if needle == "" {
		return false
	}
	return strings.Contains(hay, needle)
}

func stopWhenNeedle(goal string) string {
	stripped := stripQuoted(goal)
	lower := strings.ToLower(stripped)
	i := strings.LastIndex(lower, "stop when ")
	if i < 0 {
		return ""
	}
	rest := strings.TrimSpace(stripped[i+10:])
	for _, suffix := range []string{" is visible", " are visible", " appears", " is shown", "."} {
		if j := strings.Index(strings.ToLower(rest), suffix); j >= 0 {
			rest = rest[:j]
			break
		}
	}
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return ""
	}
	return strings.ToLower(rest)
}

func stripQuoted(s string) string {
	var b strings.Builder
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quote == 0 && (c == '"' || c == '\'') {
			quote = c
			continue
		}
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func (a *Agent) elapsed() int {
	if a.State.StartedAt.IsZero() {
		return 0
	}
	return int(time.Since(a.State.StartedAt).Milliseconds())
}

func (a *Agent) histories() []policy.History {
	out := make([]policy.History, 0, len(a.State.History))
	for _, e := range a.State.History {
		out = append(out, e.Summary())
	}
	return out
}

func (a *Agent) done(p page.Page) bool {
	fn := a.VerifyDone
	if fn == nil {
		fn = GoalVisible
	}
	return fn(a.State.Goal, p)
}

func (a *Agent) Command(name string, fingerprint string) (result error) {
	defer func() {
		if a.Coverage == nil {
			return
		}
		a.Coverage.Observe(a.State.Page)
		a.Coverage.Status = a.State.Status
		if result != nil && !IsStale(result) {
			a.Coverage.Status = "interrupted"
		}
		if err := a.saveCoverage(); err != nil {
			result = errors.Join(result, err)
		}
	}()
	switch name {
	case "tick":
		err := a.Command("predict", "")
		if err == nil {
			err = a.Command("act", a.State.Page.Fingerprint)
		}
		if err == nil {
			return nil
		}
		if !IsStale(err) {
			return err
		}
		a.State.Decision = nil
		a.State.Status = "ready"
		p, oerr := a.Surface.Observe(a.Screenshots)
		if oerr != nil {
			return oerr
		}
		a.State.Page = p
		a.State.ElapsedMS = a.elapsed()
		return nil
	case "predict":
		if a.State.StartedAt.IsZero() {
			a.State.StartedAt = time.Now()
		}
		if !a.Surface.Fresh(a.State.Page, nil) {
			p, err := a.Surface.Observe(a.Screenshots)
			if err != nil {
				return err
			}
			a.State.Page = p
		}
		a.State.Decision = nil
		if a.State.Status == "done" || a.State.Status == "blocked" {
			return fmt.Errorf("this run has stopped")
		}
		if len(a.State.Decisions) >= policy.MaxSteps*2 {
			return fmt.Errorf("reached the model-call budget")
		}
		d, err := a.Chooser.Choose(a.State.Page, a.State.Goal, a.histories())
		if err != nil {
			return err
		}
		a.State.Decision = &d
		a.State.Decisions = append(a.State.Decisions, d)
		a.State.Status = "predicted"
		a.State.ElapsedMS = a.elapsed()
		return nil
	case "act":
		d := a.State.Decision
		p := a.State.Page
		if d == nil || fingerprint != p.Fingerprint {
			return fmt.Errorf("observe and choose before acting")
		}
		a.State.Decision = nil
		if d.Choice == "DONE" || d.Choice == "BLOCKED" {
			if !a.Surface.Fresh(p, nil) {
				a.State.Status = "ready"
				return stale.Error{Msg: "page changed since the decision. Observe again"}
			}
			if d.Choice == "BLOCKED" {
				a.State.Status = "blocked"
				a.State.ElapsedMS = a.elapsed()
				return nil
			}
			if a.done(p) {
				a.State.Status = "done"
				a.State.ElapsedMS = a.elapsed()
				return nil
			}
			a.failedDone++
			if a.failedDone >= 3 {
				a.State.Status = "blocked"
				a.State.ElapsedMS = a.elapsed()
				return nil
			}
			a.State.Status = "ready"
			a.State.ElapsedMS = a.elapsed()
			return nil
		}
		action, ok := page.FindAction(p.Actions, d.Choice)
		if !ok {
			return fmt.Errorf("unknown action %s", d.Choice)
		}
		if a.done(p) {
			a.State.Status = "done"
			a.State.ElapsedMS = a.elapsed()
			return nil
		}
		if !a.Surface.Fresh(p, &action) {
			return stale.Error{Msg: "page changed since the decision. Observe again"}
		}
		if len(a.State.History) >= policy.MaxSteps {
			a.State.Status = "blocked"
			return fmt.Errorf("stopped at the %d-action budget", policy.MaxSteps)
		}
		var text *string
		var helper policy.TextHelper
		if action.Kind == "fill" {
			if !a.Surface.Fresh(p, &action) {
				return stale.Error{Msg: "page changed before text generation. Choose again"}
			}
			ctx := policy.FieldContext(a.State.Goal, action, p, a.histories())
			if a.pending != nil && mustJSON(a.pending.Context) == mustJSON(ctx) {
				t := a.pending.Text
				text, helper = &t, a.pending.Helper
			} else {
				t, h, err := a.Chooser.FieldText(ctx)
				if err != nil {
					return err
				}
				text, helper = &t, h
				a.pending = &pendingText{Context: ctx, Text: t, Helper: h}
				a.State.TextCalls = append(a.State.TextCalls, policy.TextCall{
					Field: action.Label, Value: t, Model: h.Model, LatencyMS: h.LatencyMS, Usage: h.Usage,
				})
			}
			if !a.Surface.Fresh(p, &action) {
				return stale.Error{Msg: "page changed before typing. Observe again"}
			}
		}
		if a.Coverage != nil {
			a.Coverage.Attempt(p, action)
			if err := a.saveCoverage(); err != nil {
				return err
			}
		}
		if err := a.Surface.Act(action, p, text); err != nil {
			return err
		}
		a.pending = nil
		a.failedDone = 0
		a.State.ElapsedMS = a.elapsed()
		prob := 0.0
		if d.Probabilities != nil {
			prob = d.Probabilities[d.Choice]
		}
		textVal := ""
		if text != nil {
			textVal = *text
		}
		entry := HistoryEntry{
			Outcome: "unverified", Before: p.Fingerprint,
			Step: len(a.State.History) + 1, Action: action.Label, Kind: action.Kind, Choice: d.Choice,
			Probability: prob, Confidence: d.Confidence, LatencyMS: d.LatencyMS, Text: textVal,
			TextHelper: helper.Model, TextLatencyMS: helper.LatencyMS, Operation: d.Operation, Target: d.Target,
			URL: p.URL, Usage: d.Usage, ExecutedMS: a.elapsed(), ElapsedMS: a.elapsed(),
		}
		a.State.History = append(a.State.History, entry)
		next, err := a.Surface.Observe(a.Screenshots)
		if err != nil {
			return err
		}
		changed := next.Fingerprint != p.Fingerprint
		if a.Coverage != nil {
			a.Coverage.Result(p, action, next, text)
		}
		a.State.Page = next
		a.State.ElapsedMS = a.elapsed()
		a.State.History[len(a.State.History)-1].PageChanged = &changed
		a.State.History[len(a.State.History)-1].After = next.Fingerprint
		a.State.History[len(a.State.History)-1].Outcome = ActionOutcome(action, text, p, next)
		a.State.History[len(a.State.History)-1].URL = next.URL
		a.State.History[len(a.State.History)-1].ElapsedMS = a.State.ElapsedMS
		if a.State.Record && next.Screenshot != "" {
			b, err := base64.StdEncoding.DecodeString(next.Screenshot)
			if err == nil {
				_ = os.WriteFile(filepath.Join(a.RecordDir, fmt.Sprintf("%06d.jpg", a.State.ElapsedMS)), b, 0o644)
			}
		}
		if a.done(next) {
			a.State.Status = "done"
			return nil
		}
		h := a.State.History
		if RepeatedCycle(h) {
			a.State.Status = "blocked"
			return nil
		}
		if len(h) >= 3 {
			tail := h[len(h)-3:]
			stuck := true
			for _, e := range tail {
				if e.PageChanged == nil || *e.PageChanged || e.Kind == "wait" || e.Kind == "key_left" || e.Kind == "key_right" || e.Kind == "key_select_all" || e.Kind == "key_select_left" || e.Kind == "key_select_right" {
					stuck = false
					break
				}
			}
			if stuck {
				a.State.Status = "blocked"
				return nil
			}
		}
		a.State.Status = "ready"
		return nil
	default:
		return fmt.Errorf("unknown command")
	}
}

func (a *Agent) Run() error {
	for a.State.Status != "done" && a.State.Status != "blocked" {
		before := len(a.State.History)
		if err := a.Command("tick", ""); err != nil {
			return err
		}
		if len(a.State.History) > before {
			h := a.State.History[len(a.State.History)-1]
			fmt.Printf("%5d ms  %s %s  %s  %s\n", h.ElapsedMS, h.Operation, h.Action, h.Outcome, a.State.Status)
		} else {
			fmt.Printf("%5d ms  reobserve  %s\n", a.State.ElapsedMS, a.State.Status)
		}
	}
	return nil
}

func (a *Agent) Close() error { return a.Surface.Close() }

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
