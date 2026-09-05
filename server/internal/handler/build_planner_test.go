package handler

import (
	"encoding/json"
	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"testing"
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
