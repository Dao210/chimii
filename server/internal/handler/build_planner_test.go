package handler

import (
	"encoding/json"
	"testing"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
)

func validBuildDecision() buildPlanningDecision {
	r := buildstudio.ExampleRecipe("robot")
	r.Summary = "A robot"
	r.Subject = "robot"
	return buildPlanningDecision{Outcome: "ready", Recipe: &r}
}
func TestParseBuildDecisionPreservesCompleteRecipe(t *testing.T) {
	raw, _ := json.Marshal(validBuildDecision())
	d, err := parseBuildDecision(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Recipe.Modules) != 3 || d.Question != nil {
		t.Fatalf("unexpected decision: %#v", d)
	}
}

func TestParseBuildDecisionRejectsCapabilityMetadataInModuleInstance(t *testing.T) {
	raw, _ := json.Marshal(validBuildDecision())
	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatal(err)
	}
	modules := payload["recipe"].(map[string]any)["modules"].([]any)
	modules[0].(map[string]any)["root"] = true
	raw, _ = json.Marshal(payload)
	if _, err := parseBuildDecision(string(raw)); err == nil {
		t.Fatal("library metadata was accepted as an executable module instance")
	}
}
func TestParseBuildDecisionRejectsAmbiguousAndInventedContracts(t *testing.T) {
	for _, change := range []func(*buildPlanningDecision){
		func(d *buildPlanningDecision) { d.Outcome = "maybe" },
		func(d *buildPlanningDecision) { d.Question = &buildPlanningQuestion{Prompt: "irrelevant"} },
		func(d *buildPlanningDecision) { d.Recipe.Modules[0].Kind = "castle-generator" },
		func(d *buildPlanningDecision) { d.Recipe.Modules[1].Parent = "missing" },
		func(d *buildPlanningDecision) { d.Recipe.Constraints.RequiredModules = []string{"long-ears"} },
		func(d *buildPlanningDecision) { d.Recipe.Constraints.PartCount = 37 },
	} {
		d := validBuildDecision()
		change(&d)
		raw, _ := json.Marshal(d)
		if _, err := parseBuildDecision(string(raw)); err == nil {
			t.Fatalf("accepted invalid decision: %s", raw)
		}
	}
	raw, _ := json.Marshal(validBuildDecision())
	for _, suffix := range []string{"{}", " trailing"} {
		if _, err := parseBuildDecision(string(raw) + suffix); err == nil {
			t.Fatal("accepted trailing content")
		}
	}
}
func TestParseBuildDecisionAllowsFreeTextWithoutForcedChoices(t *testing.T) {
	d := validBuildDecision()
	d.Outcome = "clarify"
	d.Recipe.Modules = nil
	d.Question = &buildPlanningQuestion{Prompt: "Which kind of apple?", Choices: []buildstudio.QuestionChoice{}}
	raw, _ := json.Marshal(d)
	got, err := parseBuildDecision(string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Question.AllowFreeText {
		t.Fatal("free text unavailable")
	}
}

func TestParseBuildDecisionRejectsExecutableClarifications(t *testing.T) {
	for _, shapeDesign := range []bool{false, true} {
		d := validBuildDecision()
		d.Outcome = "clarify"
		d.Question = &buildPlanningQuestion{Prompt: "Which size?", Choices: []buildstudio.QuestionChoice{}}
		if shapeDesign {
			d.Recipe.Version = 3
			d.Recipe.Modules = nil
			d.Recipe.Design = &buildstudio.DesignSpec{Version: 1, Mode: "static", Shapes: []buildstudio.ShapeNode{{ID: "base", Kind: "box", Operation: "add", Size: buildstudio.DesignVector{X: 6, Y: 6, Z: 6}, Color: 1}}}
		}
		if _, err := parseBuildDecision(string(mustBuildJSON(d))); err == nil {
			t.Fatalf("accepted clarification with executable content: shape_design=%v", shapeDesign)
		}
	}
}

func TestBuildAnswersAreVersionedAndIdempotent(t *testing.T) {
	q := buildstudio.ClarifyingQuestion{ID: "q2", Prompt: "Ears?", Choices: []buildstudio.QuestionChoice{{ID: "long", Label: "Long ears"}}, Options: []string{"Long ears"}, AllowFreeText: true}
	s := db.BuildSession{Revision: 2, Status: "clarifying", Question: mustBuildJSON(q), Answers: []byte(`{"q1":"dog"}`)}
	req := submitBuildAnswersRequest{Revision: 2, Answers: map[string]string{"q2": "long"}}
	answers, duplicate, err := validateBuildAnswer(s, req)
	if err != nil || duplicate || answers["q2"] != "Long ears" {
		t.Fatalf("normalize: %v %v %#v", err, duplicate, answers)
	}
	s.Status = "queued"
	s.Revision = 3
	s.Answers = mustBuildJSON(answers)
	if _, duplicate, err = validateBuildAnswer(s, req); err != nil || !duplicate {
		t.Fatal("same submission must be idempotent")
	}
	s.Status = "clarifying"
	if _, duplicate, err = validateBuildAnswer(s, submitBuildAnswersRequest{Answers: map[string]string{"q2": "long"}}); err != nil || !duplicate {
		t.Fatal("legacy duplicate must stay idempotent after next question")
	}
	req.Answers["q2"] = "different"
	if _, _, err = validateBuildAnswer(s, req); err == nil {
		t.Fatal("stale different answer accepted")
	}
	s.Status = "clarifying"
	s.Question = mustBuildJSON(buildstudio.ClarifyingQuestion{ID: "q3", Prompt: "Color?", AllowFreeText: true})
	if _, _, err = validateBuildAnswer(s, submitBuildAnswersRequest{Answers: map[string]string{"q2": "different"}}); err == nil {
		t.Fatal("old client stale question accepted")
	}
}
