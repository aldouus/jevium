package policy_test

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/aldous/jevium/internal/page"
	"github.com/aldous/jevium/internal/policy"
)

func samplePage() page.Page {
	return page.WithFingerprint(page.Page{
		URL:   "app://com.apple.springboard",
		Title: "SpringBoard",
		Text:  "Safari Settings",
		Actions: []page.Action{
			{ID: "e1", Kind: "fill", Label: "Address", Role: "textbox", Value: "", Node: "n-address"},
			{ID: "e2", Kind: "click", Label: "Open Address", Role: "textbox", Value: "", Node: "n-address"},
			{ID: "e3", Kind: "click", Label: "Safari", Role: "button", Value: "", Node: "n-safari"},
			{ID: "wait", Kind: "wait", Label: "Wait for the screen to update"},
			{ID: "home", Kind: "home", Label: "Go to Home Screen"},
		},
	})
}

func TestValidateChoiceRejectsInvalidAnswers(t *testing.T) {
	t.Parallel()
	ids := map[string]struct{}{"a": {}, "b": {}}
	cases := []policy.ChoiceAnswer{
		{Choice: "invented", Confidence: 1, Probabilities: map[string]float64{"a": 1, "b": 0}},
		{Choice: "a", Confidence: 1, Probabilities: map[string]float64{"a": math.NaN(), "b": 0}},
		{Choice: "a", Confidence: 1, Probabilities: map[string]float64{"a": 1}},
		{Choice: "a", Confidence: 1, Probabilities: map[string]float64{"a": 1, "b": -1}},
		{Choice: "b", Confidence: 1, Probabilities: map[string]float64{"a": 1, "b": 0}},
		{Choice: "a", Confidence: 5, Probabilities: map[string]float64{"a": 1, "b": 0}},
	}
	for _, c := range cases {
		if err := policy.ValidateChoice(c, ids); err == nil {
			t.Fatalf("expected invalid choice %+v", c)
		}
	}
}

func TestActionSpaceIndexesNodesByOperation(t *testing.T) {
	t.Parallel()
	elements, targets, controls := policy.ActionSpace(samplePage().Actions)
	if len(elements) != 2 {
		t.Fatalf("elements=%d", len(elements))
	}
	if strings.Join(elements[0].Operations, ",") != "TYPE_TEXT,CLICK" {
		t.Fatalf("ops=%v", elements[0].Operations)
	}
	if targets["TYPE_TEXT"]["1"].ID != "e1" {
		t.Fatalf("type target=%s", targets["TYPE_TEXT"]["1"].ID)
	}
	if targets["CLICK"]["1"].ID != "e2" {
		t.Fatalf("click target=%s", targets["CLICK"]["1"].ID)
	}
	if _, ok := controls["WAIT"]; !ok {
		t.Fatal("missing WAIT")
	}
	if _, ok := controls["HOME"]; !ok {
		t.Fatal("missing HOME")
	}
}

func TestChooseUsesMatchingTargetHeadOnly(t *testing.T) {
	var bodies []map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		bodies = append(bodies, body)
		questions := body["questions"].(map[string]any)
		op := questions["operation"].(map[string]any)
		criteria := op["criteria"].(map[string]any)
		probs := map[string]float64{}
		for k := range criteria {
			if k == "TYPE_TEXT" {
				probs[k] = 1
			} else {
				probs[k] = 0
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"model": "test",
			"answers": map[string]any{
				"operation":        map[string]any{"choice": "TYPE_TEXT", "confidence": 1.0, "probabilities": probs},
				"type_text_target": map[string]any{"choice": "1", "confidence": 1.0, "probabilities": map[string]float64{"1": 1}},
				"click_target":     map[string]any{"choice": "invented"},
			},
		})
	}))
	defer srv.Close()
	t.Setenv("TYPESAFE_API_KEY", "exported-key")
	client := policy.Client{BaseURL: srv.URL, HTTP: srv.Client()}
	d, err := client.Choose(samplePage(), "Search example.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(bodies) != 1 {
		t.Fatalf("calls=%d", len(bodies))
	}
	if d.Operation != "TYPE_TEXT" || d.Choice != "e1" {
		t.Fatalf("decision=%+v", d)
	}
	questions := bodies[0]["questions"].(map[string]any)
	if _, ok := questions["click_target"]; !ok {
		t.Fatal("expected speculative click_target")
	}
	if r := srv.Client(); r == nil {
		t.Fatal("client")
	}
}

func TestChooseRequiresAPIKey(t *testing.T) {
	t.Setenv("TYPESAFE_API_KEY", "")
	os.Unsetenv("TYPESAFE_API_KEY")
	_, err := policy.Client{}.Choose(samplePage(), "goal", nil)
	if err == nil || !strings.Contains(err.Error(), "TYPESAFE_API_KEY") {
		t.Fatalf("err=%v", err)
	}
}

func TestFieldTextUsesQuotedLiteralFromGoal(t *testing.T) {
	os.Unsetenv("TEXT_MODEL_API_KEY")
	text, helper, err := policy.Client{}.FieldText(policy.FieldInput{Goal: `Open google.com and type "hello world".`})
	if err != nil {
		t.Fatal(err)
	}
	if text != "hello world" {
		t.Fatalf("text=%s", text)
	}
	if helper.Model != "goal-quote" {
		t.Fatalf("helper=%v", helper)
	}
}

func TestFieldTextUsesLiteralURLFromGoal(t *testing.T) {
	os.Unsetenv("TEXT_MODEL_API_KEY")
	text, helper, err := policy.Client{}.FieldText(policy.FieldInput{Goal: "Open Safari and load https://example.com. Stop."})
	if err != nil {
		t.Fatal(err)
	}
	if text != "https://example.com" {
		t.Fatalf("text=%s", text)
	}
	if helper.Model != "goal-url" {
		t.Fatalf("helper=%v", helper)
	}
}

func TestFieldTextRequiresCredential(t *testing.T) {
	os.Unsetenv("TEXT_MODEL_API_KEY")
	_, _, err := policy.Client{}.FieldText(policy.FieldInput{Goal: "Enter Zurich"})
	if err == nil || !strings.Contains(err.Error(), "TEXT_MODEL_API_KEY") {
		t.Fatalf("err=%v", err)
	}
}

func TestFieldTextRejectsInvalidValues(t *testing.T) {
	t.Setenv("TEXT_MODEL_API_KEY", "test")
	for _, content := range []string{"Thinking: Zurich", `{"text":null}`, `{"text":"Zurich","extra":true}`, `{"text":123}`} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"choices": []map[string]any{{"message": map[string]any{"content": content}}},
			})
		}))
		client := policy.Client{TextBaseURL: srv.URL, HTTP: srv.Client()}
		_, _, err := client.FieldText(policy.FieldInput{Goal: "Find a flight"})
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), "nothing typed") {
			t.Fatalf("content=%s err=%v", content, err)
		}
	}
}
