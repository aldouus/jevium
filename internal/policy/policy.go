package policy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aldous/jevium/internal/env"
	"github.com/aldous/jevium/internal/page"
)

type ChoiceAnswer struct {
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

type Element struct {
	Index      string          `json:"index"`
	Label      string          `json:"label"`
	Role       string          `json:"role,omitempty"`
	Value      string          `json:"value,omitempty"`
	Checked    string          `json:"checked,omitempty"`
	Selected   any             `json:"selected,omitempty"`
	Expanded   string          `json:"expanded,omitempty"`
	Operations []string        `json:"operations"`
	Options    []ElementOption `json:"options,omitempty"`
}

type ElementOption struct {
	Index string `json:"index"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type Decision struct {
	Choice                 string                     `json:"choice"`
	Operation              string                     `json:"operation"`
	Target                 string                     `json:"target,omitempty"`
	Confidence             float64                    `json:"confidence"`
	Probabilities          map[string]float64         `json:"probabilities"`
	OperationProbabilities map[string]float64         `json:"operation_probabilities"`
	TargetProbabilities    map[string]float64         `json:"target_probabilities"`
	TargetConfidence       *float64                   `json:"target_confidence"`
	RawAnswers             map[string]json.RawMessage `json:"raw_answers"`
	Model                  string                     `json:"model"`
	Usage                  map[string]any             `json:"usage"`
	LatencyMS              int                        `json:"latency_ms"`
}

type History struct {
	Action      string `json:"action"`
	Kind        string `json:"kind"`
	Text        string `json:"text,omitempty"`
	PageChanged *bool  `json:"page_changed,omitempty"`
}

type Field struct {
	Label string `json:"label"`
	Role  string `json:"role,omitempty"`
	Value string `json:"value,omitempty"`
}

type PageView struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

type FieldInput struct {
	Goal          string    `json:"goal"`
	Field         Field     `json:"field"`
	Page          PageView  `json:"page"`
	RecentActions []History `json:"recent_actions"`
}

type TextHelper struct {
	Model     string         `json:"model"`
	LatencyMS int            `json:"latency_ms"`
	Usage     map[string]any `json:"usage,omitempty"`
}

type TextCall struct {
	Field     string         `json:"field"`
	Value     string         `json:"value"`
	Model     string         `json:"model"`
	LatencyMS int            `json:"latency_ms"`
	Usage     map[string]any `json:"usage,omitempty"`
}

type Client struct {
	BaseURL     string
	TextBaseURL string
	HTTP        *http.Client
}

func (c Client) http() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return &http.Client{Timeout: 60 * time.Second}
}

func (c Client) systemOneURL() string {
	if c.BaseURL != "" {
		return strings.TrimRight(c.BaseURL, "/") + "/v1/systemone"
	}
	return "https://api.typesafe.ai/v1/systemone"
}

func (c Client) textURL() string {
	base := c.TextBaseURL
	if base == "" {
		base = env.Get("TEXT_MODEL_BASE_URL", "https://api.deepseek.com/v1")
	}
	return strings.TrimRight(base, "/") + "/chat/completions"
}

func ValidateChoice(answer ChoiceAnswer, ids map[string]struct{}) error {
	if _, ok := ids[answer.Choice]; !ok {
		return fmt.Errorf("invalid TypeSafe response; no action executed")
	}
	if len(answer.Probabilities) != len(ids) {
		return fmt.Errorf("invalid TypeSafe response; no action executed")
	}
	max := math.Inf(-1)
	sum := 0.0
	for id := range ids {
		n, ok := answer.Probabilities[id]
		if !ok || math.IsNaN(n) || math.IsInf(n, 0) || n < 0 || n > 1 {
			return fmt.Errorf("invalid TypeSafe response; no action executed")
		}
		sum += n
		if n > max {
			max = n
		}
	}
	if answer.Confidence < 0 || answer.Confidence > 1 || math.IsNaN(answer.Confidence) || math.IsInf(answer.Confidence, 0) {
		return fmt.Errorf("invalid TypeSafe response; no action executed")
	}
	if math.Abs(sum-1) >= 0.02 {
		return fmt.Errorf("invalid TypeSafe response; no action executed")
	}
	if answer.Probabilities[answer.Choice] < max-1e-6 {
		return fmt.Errorf("invalid TypeSafe response; no action executed")
	}
	return nil
}

var kindOps = map[string]string{
	"key_select_all": "SELECT_ALL", "key_select_left": "SELECT_LEFT", "key_select_right": "SELECT_RIGHT",
	"clear_text":    "CLEAR_TEXT",
	"key_backspace": "BACKSPACE",
	"key_left":      "CURSOR_LEFT",
	"key_right":     "CURSOR_RIGHT",
	"key_return":    "RETURN",
	"click":         "CLICK",
	"fill":          "TYPE_TEXT",
	"select":        "SELECT",
}

func ActionSpace(actions []page.Action) ([]Element, map[string]map[string]page.Action, map[string]page.Action) {
	elements := []Element{}
	indices := map[string]string{}
	targets := map[string]map[string]page.Action{}
	controls := map[string]page.Action{}
	for _, action := range actions {
		op, ok := kindOps[action.Kind]
		if !ok {
			controls[strings.ToUpper(action.ID)] = action
			continue
		}
		nodeKey := fmt.Sprint(action.Node)
		index, seen := indices[nodeKey]
		if !seen {
			index = fmt.Sprintf("%d", len(elements)+1)
			indices[nodeKey] = index
			el := Element{
				Index:      index,
				Label:      strings.Split(action.Label, " → ")[0],
				Role:       action.Role,
				Value:      action.Value,
				Checked:    action.Checked,
				Selected:   action.Selected,
				Expanded:   action.Expanded,
				Operations: []string{},
			}
			if action.Kind == "select" {
				el.Value = action.CurrentValue
				el.Options = []ElementOption{}
			}
			elements = append(elements, el)
		}
		group, ok := targets[op]
		if !ok {
			group = map[string]page.Action{}
			targets[op] = group
		}
		el := &elements[len(elements)-1]
		if index != el.Index {
			for i := range elements {
				if elements[i].Index == index {
					el = &elements[i]
					break
				}
			}
		}
		if !contains(el.Operations, op) {
			el.Operations = append(el.Operations, op)
		}
		target := index
		if action.Kind == "select" {
			target = fmt.Sprintf("%s:%d", index, len(el.Options)+1)
			el.Options = append(el.Options, ElementOption{Index: target, Label: action.Label, Value: action.Value})
		}
		group[target] = action
	}
	return elements, targets, controls
}

func contains(xs []string, v string) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}

func (c Client) Choose(state page.Page, goal string, history []History) (Decision, error) {
	key, err := env.Require("TYPESAFE_API_KEY", "to call TypeSafe Jev")
	if err != nil {
		return Decision{}, err
	}
	elements, targets, controls := ActionSpace(state.Actions)
	labels := map[string]string{
		"SELECT_ALL":   "Select all text in the focused field using Command+A.",
		"SELECT_LEFT":  "Extend the focused text selection one character left using Shift+Left.",
		"SELECT_RIGHT": "Extend the focused text selection one character right using Shift+Right.",
		"CLEAR_TEXT":   "Clear all text in the observed field.",
		"BACKSPACE":    "Delete one character before the focused cursor.",
		"CURSOR_LEFT":  "Move the focused cursor one character left.",
		"CURSOR_RIGHT": "Move the focused cursor one character right.",
		"RETURN":       "Press Return in the focused field; this may submit the form.",
		"CLICK":        "Click an element, button, menu option, autocomplete suggestion, or calendar day.",
		"TYPE_TEXT":    "Enter or replace text in an editable field. A small LLM will supply the value from the goal.",
		"SELECT":       "Select an observed dropdown value.",
	}
	operations := map[string]any{}
	for key := range targets {
		operations[key] = labels[key]
	}
	for key, value := range controls {
		operations[key] = value.Label
	}
	operations["DONE"] = "Every requirement is visibly satisfied."
	operations["BLOCKED"] = "No supported operation can make progress."
	questions := map[string]any{
		"operation": map[string]any{
			"type":         "choice",
			"criteria":     operations,
			"instructions": map[string]any{"goal": goal, "rules": NextAction},
		},
	}
	for operation, candidates := range targets {
		criteria := map[string]any{}
		for index, a := range candidates {
			item := map[string]any{
				"element":       fmt.Sprintf("[%s] %s", index, a.Label),
				"current_value": firstNonEmpty(a.CurrentValue, a.Value),
			}
			if a.Role != "" {
				item["role"] = a.Role
			}
			if a.Checked != "" {
				item["checked"] = a.Checked
			}
			if a.Selected != nil {
				item["selected"] = a.Selected
			}
			if a.Expanded != "" {
				item["expanded"] = a.Expanded
			}
			criteria[index] = item
		}
		questions[strings.ToLower(operation)+"_target"] = map[string]any{
			"type":     "choice",
			"criteria": criteria,
			"instructions": map[string]any{
				"goal":      goal,
				"operation": operation,
				"rules":     []string{NextAction, Target},
			},
		}
	}
	recent := history
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	body := map[string]any{
		"model": env.Get("TYPESAFE_MODEL", "jev-latest"),
		"state": map[string]any{
			"page": map[string]any{
				"url":   state.URL,
				"title": state.Title,
				"text":  state.Text,
			},
			"elements":       elements,
			"recent_actions": recent,
		},
		"questions": questions,
	}
	started := time.Now()
	raw, err := c.postJSON(c.systemOneURL(), key, body)
	if err != nil {
		return Decision{}, err
	}
	var result struct {
		Model   string                     `json:"model"`
		Answers map[string]json.RawMessage `json:"answers"`
		Usage   map[string]any             `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return Decision{}, err
	}
	opIDs := map[string]struct{}{}
	for k := range operations {
		opIDs[k] = struct{}{}
	}
	opAnswer, err := decodeChoice(result.Answers["operation"])
	if err != nil {
		return Decision{}, err
	}
	if err := ValidateChoice(opAnswer, opIDs); err != nil {
		return Decision{}, err
	}
	d := Decision{
		Operation:              opAnswer.Choice,
		Confidence:             opAnswer.Confidence,
		OperationProbabilities: opAnswer.Probabilities,
		RawAnswers:             result.Answers,
		Model:                  result.Model,
		Usage:                  result.Usage,
		LatencyMS:              int(time.Since(started).Milliseconds()),
		Probabilities:          map[string]float64{},
		TargetProbabilities:    map[string]float64{},
	}
	if candidates, ok := targets[opAnswer.Choice]; ok {
		ids := map[string]struct{}{}
		for k := range candidates {
			ids[k] = struct{}{}
		}
		targetAnswer, err := decodeChoice(result.Answers[strings.ToLower(opAnswer.Choice)+"_target"])
		if err != nil {
			return Decision{}, err
		}
		if err := ValidateChoice(targetAnswer, ids); err != nil {
			return Decision{}, err
		}
		d.Target = targetAnswer.Choice
		d.Choice = candidates[targetAnswer.Choice].ID
		conf := targetAnswer.Confidence
		d.TargetConfidence = &conf
		d.TargetProbabilities = targetAnswer.Probabilities
		d.Probabilities = map[string]float64{}
		for index, a := range candidates {
			d.Probabilities[a.ID] = targetAnswer.Probabilities[index]
		}
	} else if control, ok := controls[opAnswer.Choice]; ok {
		d.Choice = control.ID
		d.Probabilities[control.ID] = opAnswer.Probabilities[opAnswer.Choice]
	} else {
		d.Choice = opAnswer.Choice
		d.Probabilities[opAnswer.Choice] = opAnswer.Probabilities[opAnswer.Choice]
	}
	return d, nil
}

func decodeChoice(raw json.RawMessage) (ChoiceAnswer, error) {
	var a ChoiceAnswer
	if len(raw) == 0 {
		return a, fmt.Errorf("invalid TypeSafe response; no action executed")
	}
	if err := json.Unmarshal(raw, &a); err != nil {
		return a, fmt.Errorf("invalid TypeSafe response; no action executed")
	}
	return a, nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func FieldContext(goal string, action page.Action, p page.Page, history []History) FieldInput {
	text := p.Text
	if len(text) > 6000 {
		text = text[:6000]
	}
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	recent := make([]History, 0, len(history[start:]))
	for _, h := range history[start:] {
		recent = append(recent, History{Action: h.Action, Text: h.Text})
	}
	return FieldInput{
		Goal:          goal,
		Field:         Field{Label: action.Label, Role: action.Role, Value: action.Value},
		Page:          PageView{Title: p.Title, Text: text},
		RecentActions: recent,
	}
}

func LiteralQuote(goal string) string {
	for _, q := range []byte{'"', '\''} {
		start := strings.IndexByte(goal, q)
		if start < 0 {
			continue
		}
		rest := goal[start+1:]
		end := strings.IndexByte(rest, q)
		if end <= 0 {
			continue
		}
		return rest[:end]
	}
	return ""
}

func LiteralURL(goal string) string {
	for _, prefix := range []string{"https://", "http://"} {
		i := strings.Index(strings.ToLower(goal), prefix)
		if i < 0 {
			continue
		}
		rest := goal[i:]
		end := len(rest)
		for j, r := range rest {
			if r == ' ' || r == '"' || r == '\'' || r == ',' || r == ';' || r == ')' || r == ']' {
				end = j
				break
			}
		}
		u := strings.TrimRight(rest[:end], ".)")
		if len(u) > len(prefix) {
			return u
		}
	}
	return ""
}

func (c Client) FieldText(context FieldInput) (string, TextHelper, error) {
	if context.Goal != "" {
		if q := LiteralQuote(context.Goal); q != "" {
			return q, TextHelper{Model: "goal-quote"}, nil
		}
		if u := LiteralURL(context.Goal); u != "" {
			return u, TextHelper{Model: "goal-url"}, nil
		}
	}
	key := strings.TrimSpace(os.Getenv("TEXT_MODEL_API_KEY"))
	if key == "" {
		return "", TextHelper{}, fmt.Errorf("TYPE_TEXT needs TEXT_MODEL_API_KEY; no text is hardcoded or guessed by the executor")
	}
	modelName := env.Get("TEXT_MODEL", "deepseek-chat")
	payload := map[string]any{
		"model":           modelName,
		"max_tokens":      1024,
		"response_format": map[string]any{"type": "json_object"},
		"messages": []map[string]any{
			{"role": "system", "content": TextValue},
			{"role": "user", "content": mustJSON(context)},
		},
	}
	base := c.TextBaseURL
	if base == "" {
		base = env.Get("TEXT_MODEL_BASE_URL", "https://api.deepseek.com/v1")
	}
	if strings.Contains(base, "api.deepseek.com/") {
		payload["thinking"] = map[string]any{"type": "disabled"}
	} else if os.Getenv("TEXT_MODEL_REASONING") == "none" {
		payload["reasoning"] = map[string]any{"enabled": false}
	} else {
		payload["reasoning"] = map[string]any{"effort": "low"}
	}
	started := time.Now()
	raw, err := c.postJSON(c.textURL(), key, payload)
	if err != nil {
		return "", TextHelper{}, err
	}
	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage map[string]any `json:"usage"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.Choices) == 0 {
		return "", TextHelper{}, fmt.Errorf("text helper returned no valid field value; nothing typed")
	}
	var output map[string]any
	if err := json.Unmarshal([]byte(result.Choices[0].Message.Content), &output); err != nil {
		return "", TextHelper{}, fmt.Errorf("text helper returned no valid field value; nothing typed")
	}
	if len(output) != 1 {
		return "", TextHelper{}, fmt.Errorf("text helper returned no valid field value; nothing typed")
	}
	text, ok := output["text"].(string)
	if !ok || strings.TrimSpace(text) == "" || len(text) > 2000 {
		return "", TextHelper{}, fmt.Errorf("text helper returned no valid field value; nothing typed")
	}
	return text, TextHelper{
		Model:     modelName,
		LatencyMS: int(time.Since(started).Milliseconds()),
		Usage:     result.Usage,
	}, nil
}

func mustJSON(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (c Client) postJSON(url, key string, body any) ([]byte, error) {
	raw, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+key)
		req.Header.Set("Content-Type", "application/json")
		client := c.http()
		resp, err := client.Do(req)
		if err != nil {
			if attempt < 2 {
				time.Sleep(time.Duration(400*(1<<attempt)) * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("model connection failed; no action executed: %w", err)
		}
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode == 429 || resp.StatusCode == 529 || resp.StatusCode == 503 {
			last = fmt.Errorf("model provider returned HTTP %d; no action executed", resp.StatusCode)
			time.Sleep(time.Duration(500*(1<<attempt)) * time.Millisecond)
			continue
		}
		if resp.StatusCode >= 400 {
			return nil, fmt.Errorf("model provider returned HTTP %d; no action executed", resp.StatusCode)
		}
		return data, nil
	}
	if last != nil {
		return nil, last
	}
	return nil, fmt.Errorf("model unavailable")
}
