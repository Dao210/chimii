package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"strings"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/pkg/llm"
)

const buildIntentTimeout = 20 * time.Second

// A complete shape recipe takes longer than a routing decision. Keep this
// within the worker lease; the utility client honors this explicit context.
const buildRecipeTimeout = 120 * time.Second
const buildPlannerTimeoutCode = "BUILD_PLANNER_TIMEOUT"
const maxBuildClarifications = 2

type buildPlanningDecision struct {
	Outcome  string                      `json:"outcome"`
	Message  string                      `json:"message,omitempty"`
	Recipe   *buildstudio.AssemblyRecipe `json:"recipe,omitempty"`
	Question *buildPlanningQuestion      `json:"question,omitempty"`
}
type buildPlanningQuestion struct {
	Prompt        string                       `json:"prompt"`
	Choices       []buildstudio.QuestionChoice `json:"choices"`
	AllowFreeText bool                         `json:"allow_free_text"`
}

func (h *Handler) planBuildRecipe(ctx context.Context, prompt string, answers map[string]string, draft *buildstudio.AssemblyRecipe, revision int32, inventory buildstudio.InventorySnapshot, catalog buildstudio.PartCatalog, conversation ...[]map[string]string) (buildPlanningDecision, error) {
	if h.LLM == nil || !h.LLM.Enabled() {
		return buildPlanningDecision{}, errors.New("build planner is not configured")
	}
	input := struct {
		Idea               string                          `json:"idea"`
		History            map[string]string               `json:"history"`
		Draft              *buildstudio.AssemblyRecipe     `json:"draft,omitempty"`
		QuestionsRemaining int                             `json:"questions_remaining"`
		Capabilities       any                             `json:"capabilities"`
		Colors             []int                           `json:"colors"`
		ColorNames         map[int]string                  `json:"color_names"`
		Constructions      any                             `json:"constructions"`
		MaxSpan            int                             `json:"max_span_studs"`
		Conversation       [][]map[string]string           `json:"conversation"`
		CatalogProfile     buildstudio.SceneCatalogProfile `json:"catalog_profile"`
	}{prompt, answers, draft, max(0, maxBuildClarifications-int(revision)+1), nil, buildstudio.AllowedColorCodes(), buildstudio.SceneColorNames(), buildstudio.ConstructionCapabilities(), buildSceneMaxSpan(catalog), conversation, buildstudio.DescribeSceneCatalog(catalog)}
	if draft != nil && draft.Version < 4 {
		input.Capabilities = buildstudio.Capabilities(buildstudio.UnlimitedInventory(), catalog)
	}
	rawInput, err := json.Marshal(input)
	if err != nil {
		return buildPlanningDecision{}, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, buildRecipeTimeout)
	defer cancel()
	// Relative anchors and constructions own coordinate arithmetic. Bounded
	// reasoning is still needed to choose compatible functions and interfaces.
	raw, err := h.LLM.GenerateContent(requestCtx, "", buildSceneSystemPrompt, string(rawInput), llm.GenerationOptions{Effort: h.buildPlanningEffort(), Purpose: "build-plan"})
	if err != nil {
		// A full planner timeout is terminal for this turn. Repeating the same
		// long request three times hides the failure and compounds provider load.
		// Shutdown/caller cancellation must retain its existing retry semantics.
		if ctx.Err() == nil && errors.Is(err, context.DeadlineExceeded) {
			return buildPlanningDecision{}, &buildstudio.BuildError{Code: buildPlannerTimeoutCode, Cause: err}
		}
		return buildPlanningDecision{}, err
	}
	parse := func(raw string) (buildPlanningDecision, error) {
		decision, err := parseGeneratedBuildDecision(raw)
		if err != nil || decision.Outcome != "ready" || decision.Recipe.Scene == nil {
			return decision, err
		}
		decision.Recipe.Prompt = prompt
		if draft == nil {
			if decision.Recipe.Scene.Version != 2 {
				return decision, errors.New("new scenes require version 2 with subject roles and assembly interfaces")
			}
			for _, object := range decision.Recipe.Scene.Objects {
				for _, module := range object.Modules {
					if module.Attach == nil {
						return decision, fmt.Errorf("new construction %s/%s needs a typed relative attachment", object.ID, module.ID)
					}
				}
			}
		}
		return decision, buildstudio.ValidateScene(*decision.Recipe)
	}
	decision, err := parse(raw)
	if err != nil {
		// Correct malformed model output once within this turn, rather than
		// replaying the whole planning request through the job retry loop.
		cancel()
		repairInput, _ := json.Marshal(map[string]any{"request": input, "invalid_output": raw, "validation_error": err.Error()})
		repairCtx, repairCancel := context.WithTimeout(ctx, 90*time.Second)
		raw, err = h.LLM.GenerateContent(repairCtx, "", buildSceneSystemPrompt+"\nCorrect every validation error and return one complete valid planning decision. Preserve the user's explicit features. Check all modules before answering, not just the first diagnostic.", string(repairInput), llm.GenerationOptions{Effort: h.buildPlanningEffort(), Purpose: "build-plan-correction"})
		repairCancel()
		if err == nil {
			decision, err = parse(raw)
		}
		if err != nil {
			return decision, &buildstudio.BuildError{Code: buildstudio.BuildErrorRequirements, Cause: err}
		}
	}
	if decision.Outcome == "clarify" && revision > maxBuildClarifications {
		return buildPlanningDecision{}, &buildstudio.BuildError{Code: buildstudio.BuildErrorUnsupported, Cause: errors.New("clarification budget exhausted")}
	}
	if decision.Recipe != nil {
		recipe := decision.Recipe
		recipe.Prompt = prompt
		recipe.Archetype = recipe.Subject
		recipe.Palette = []int{}
		recipe.Features = []string{}
		sizing := recipe.Metadata["construction_sizing"]
		recipe.Metadata = map[string]string{"planner": "llm-shapes-v1"}
		if sizing != "" {
			recipe.Metadata["construction_sizing"] = sizing
		}
		if recipe.Scene != nil {
			recipe.Metadata["planner"] = buildstudio.SceneGeneratorVersion
		}
		// Exact counts from the source request cannot disappear in model output.
		policySource := *recipe
		if recipe.Scene != nil {
			// The legacy default target (80) is not the scene's automatic scale.
			// Apply only source-specified exact counts to a new scene.
			policySource.Metadata = maps.Clone(recipe.Metadata)
		}
		policy := buildstudio.ApplyDifficultyPolicy(policySource, prompt, answers)
		if policy.Metadata["part_count_source"] == "explicit" {
			recipe.Metadata = policy.Metadata
			recipe.Constraints.PartCount = buildstudio.RequestedPartCount(policy)
		}
		if draft != nil {
			for _, key := range []string{"parent_creation_id", "parent_hash"} {
				if value := draft.Metadata[key]; value != "" {
					recipe.Metadata[key] = value
				}
			}
			if draft.Constraints.NoWheels {
				recipe.Constraints.NoWheels = true
			}
			if draft.Constraints.ExactColors {
				recipe.Constraints.ExactColors = true
			}
			if draft.Constraints.PartCount > 0 && policy.Metadata["part_count_source"] != "explicit" {
				recipe.Constraints.PartCount = draft.Constraints.PartCount
			}
			for _, kind := range draft.Constraints.RequiredModules {
				if !buildContainsString(recipe.Constraints.RequiredModules, kind) {
					recipe.Constraints.RequiredModules = append(recipe.Constraints.RequiredModules, kind)
				}
			}
		}
	}
	return decision, nil
}

func buildContainsString(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func parseBuildDecision(raw string) (buildPlanningDecision, error) {
	return parseBuildDecisionWithSizing(raw, false)
}
func parseGeneratedBuildDecision(raw string) (buildPlanningDecision, error) {
	return parseBuildDecisionWithSizing(raw, true)
}
func parseBuildDecisionWithSizing(raw string, fit bool) (buildPlanningDecision, error) {
	var d buildPlanningDecision
	if len(raw) > 96<<10 {
		return d, errors.New("build decision too large")
	}
	decoder := json.NewDecoder(strings.NewReader(buildModelJSON(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&d); err != nil {
		return d, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return d, errors.New("build decision contains trailing content")
	}
	if fit && d.Recipe != nil && d.Outcome == "ready" {
		r := buildstudio.FitGeneratedScene(*d.Recipe)
		d.Recipe = &r
	}
	textValid := func(s string, n int) bool { return strings.TrimSpace(s) != "" && len([]rune(s)) <= n }
	if d.Recipe != nil {
		r := d.Recipe
		maxParts := 200
		if r.Version == 4 {
			maxParts = buildstudio.MaxSceneParts
		}
		if (r.Version != 2 && r.Version != 3 && r.Version != 4) || !textValid(r.Subject, 80) || !textValid(r.Summary, 240) || !textValid(r.Title, 24) || len(r.Requirements) > 24 || len(r.Constraints.RequiredModules) > 24 || r.Constraints.PartCount < 0 || r.Constraints.PartCount > maxParts {
			return d, errors.New("invalid build recipe summary or constraints")
		}
		for _, requirement := range r.Requirements {
			if !textValid(requirement, 160) {
				return d, errors.New("invalid requirement")
			}
		}
	}
	switch d.Outcome {
	case "ready":
		if d.Question != nil || d.Recipe == nil || (len(d.Recipe.Modules) == 0 && d.Recipe.Design == nil && d.Recipe.Scene == nil) {
			return d, errors.New("ready decision needs only a complete recipe")
		}
		// Shape validation here is cheap; mechanics and inventory are checked by the compiler.
		if d.Recipe.Scene != nil {
			if err := buildstudio.ValidateScene(*d.Recipe); err != nil {
				return d, err
			}
		} else if d.Recipe.Design != nil {
			if d.Recipe.Version != 3 || len(d.Recipe.Modules) != 0 || len(d.Recipe.Constraints.RequiredModules) != 0 {
				return d, errors.New("invalid mixed design")
			}
			if _, err := buildstudio.RasterizeDesign(*d.Recipe.Design); err != nil {
				return d, err
			}
		} else {
			if _, err := buildstudio.ExpandRecipe(*d.Recipe); err != nil {
				return d, err
			}
		}
	case "clarify":
		if d.Question == nil || d.Recipe == nil || d.Recipe.Design != nil || d.Recipe.Scene != nil || len(d.Recipe.Modules) > 0 || !textValid(d.Question.Prompt, 180) || len(d.Question.Choices) > 3 {
			return d, errors.New("invalid clarification")
		}
		seen := map[string]bool{}
		labels := map[string]bool{}
		for _, choice := range d.Question.Choices {
			if !textValid(choice.ID, 32) || !textValid(choice.Label, 120) || seen[choice.ID] || labels[choice.Label] {
				return d, errors.New("invalid question choices")
			}
			seen[choice.ID] = true
			labels[choice.Label] = true
		}
		d.Question.AllowFreeText = true
	case "reply":
		if d.Recipe != nil || d.Question != nil || !textValid(d.Message, 240) {
			return d, errors.New("reply needs only a message")
		}
	case "unsupported":
		if d.Question != nil || !textValid(d.Message, 240) {
			return d, errors.New("unsupported decision needs explanation")
		}
	default:
		return d, fmt.Errorf("unknown planning outcome")
	}
	return d, nil
}

// Only unwrap one complete JSON code block at the external model boundary.
// Never extract a JSON substring from surrounding prose or ignore trailing data.
func buildModelJSON(raw string) string {
	value := strings.TrimSpace(raw)
	for _, prefix := range []string{"```json\n", "```\n"} {
		if strings.HasPrefix(value, prefix) && strings.HasSuffix(value, "```") {
			return strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, prefix), "```"))
		}
	}
	return value
}
