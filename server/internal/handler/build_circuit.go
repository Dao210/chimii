package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/circuit"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type circuitBuildPlan struct {
	SourceKind          string `json:"source_kind,omitempty"`
	AwaitingMaterials   bool   `json:"awaiting_materials,omitempty"`
	KitID               string `json:"kit_id"`
	CatalogVersion      string `json:"catalog_version"`
	InventoryRevision   int32  `json:"inventory_revision"`
	ProjectID           string `json:"project_id,omitempty"`
	Locale              string `json:"locale,omitempty"`
	SourceCreationID    string `json:"source_creation_id,omitempty"`
	ExpectedContentHash string `json:"expected_content_hash,omitempty"`
	PreviousProjectID   string `json:"previous_project_id,omitempty"`
	Title               string `json:"title,omitempty"`
	Summary             string `json:"summary,omitempty"`
}

func (h *Handler) validateCircuitBuildSource(w http.ResponseWriter, r *http.Request, q *db.Queries, ws, user, child pgtype.UUID, p *circuitBuildPlan) bool {
	id, ok := parseUUIDOrBadRequest(w, p.SourceCreationID, "source_creation_id")
	if !ok {
		return false
	}
	row, err := q.GetCircuitCreation(r.Context(), db.GetCircuitCreationParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child)})
	if err != nil {
		writeError(w, 404, "source creation not found")
		return false
	}
	var doc circuit.Document
	if json.Unmarshal(row.Document, &doc) != nil {
		writeError(w, 500, "invalid source creation")
		return false
	}
	if p.ExpectedContentHash == "" || p.ExpectedContentHash != doc.ContentHash {
		writeError(w, 409, "source content changed")
		return false
	}
	if p.KitID == "" {
		p.KitID = doc.KitID
		p.CatalogVersion = doc.CatalogVersion
	}
	if p.KitID == doc.KitID {
		p.PreviousProjectID = doc.Project.ID
	}
	return true
}

func (h *Handler) answerBuildConversation(w http.ResponseWriter, r *http.Request, q *db.Queries, tx pgx.Tx, req createBuildSessionRequest, s db.BuildSession, hash string) {
	// Lock the active run after the root, matching worker session fencing.
	current, err := q.LockBuildSessionForAnswer(r.Context(), db.LockBuildSessionForAnswerParams{ID: s.ID, WorkspaceID: s.WorkspaceID, CreatorUserID: s.CreatorUserID, ChildProfileID: s.ChildProfileID})
	if err != nil {
		writeError(w, 409, "conversation changed")
		return
	}
	answers, duplicate, err := validateBuildAnswer(current, submitBuildAnswersRequest{Revision: req.ExpectedRevision, Answers: map[string]string{req.QuestionID: req.Prompt}})
	if err != nil || duplicate {
		writeError(w, 409, "answer does not match the current question")
		return
	}
	if s.Kind != "auto" && req.Kind != s.Kind {
		writeError(w, 409, "cancel the current question before changing creation type")
		return
	}
	if s.Kind == "circuit" || s.Kind == "auto" {
		var plan circuitBuildPlan
		if json.Unmarshal(current.Recipe, &plan) != nil {
			writeError(w, 500, "invalid circuit plan")
			return
		}
		if plan.AwaitingMaterials && req.Prompt != "materials_ready" && (req.Circuit == nil || req.Circuit.ProjectID == "") {
			plan.ProjectID = ""
			plan.Title = ""
		}
		plan.AwaitingMaterials = false
		if req.Circuit != nil {
			if req.Circuit.KitID != plan.KitID {
				plan.PreviousProjectID = ""
			}
			plan.KitID, plan.CatalogVersion, plan.InventoryRevision, plan.Locale = req.Circuit.KitID, req.Circuit.CatalogVersion, req.Circuit.InventoryRevision, req.Circuit.Locale
			if req.Circuit.ProjectID != "" {
				if plan.ProjectID != req.Circuit.ProjectID {
					plan.Title = ""
				}
				plan.ProjectID = req.Circuit.ProjectID
			}
		}
		if s.Kind == "auto" && req.Kind != "auto" {
			s.Kind = req.Kind
		}
		var recipe []byte
		if s.Kind != "brick" {
			recipe = mustBuildJSON(plan)
		} else {
			sourceSession := s
			sourceSession.Recipe = mustBuildJSON(plan)
			recipe, err = routedBuildRecipe(r.Context(), q, sourceSession, "brick")
			if err != nil {
				writeError(w, 409, "could not restore source design")
				return
			}
		}
		if _, err = q.SetBuildSessionKind(r.Context(), db.SetBuildSessionKindParams{ID: s.ID, Kind: s.Kind, Recipe: recipe}); err != nil {
			writeError(w, 500, "could not save context")
			return
		}
	}
	updated, err := q.SubmitBuildSessionAnswers(r.Context(), db.SubmitBuildSessionAnswersParams{ID: s.ID, WorkspaceID: s.WorkspaceID, CreatorUserID: s.CreatorUserID, ChildProfileID: s.ChildProfileID, Revision: current.Revision, Answers: mustBuildJSON(answers)})
	if err == nil {
		err = appendBuildUserMessage(r.Context(), q, updated, req.Prompt, "request:"+req.ClientRequestID, hash)
	}
	if err == nil {
		_, err = q.EnqueueBuildJob(r.Context(), db.EnqueueBuildJobParams{WorkspaceID: s.WorkspaceID, SessionID: s.ID})
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		writeError(w, 500, "could not save answer")
		return
	}
	h.BuildWorker.Notify()
	writeJSON(w, 202, toBuildSessionResponse(updated))
}

func compileCircuitDocument(c circuit.Catalog, req circuitCreateRequest, title, planner string) (circuit.Document, error) {
	doc, err := circuit.Compile(c, req.ProjectID, req.Prompt, title, planner, req.Inventory)
	if err != nil {
		return doc, err
	}
	if req.InventoryRevision != nil {
		doc.InventoryRevision = req.InventoryRevision
		return circuit.SealDocument(doc)
	}
	return doc, nil
}

var errCircuitInventoryChanged = errors.New("CIRCUIT_INVENTORY_CHANGED")

func saveCircuitDocument(ctx context.Context, q *db.Queries, req circuitCreateRequest, params db.CreateCircuitCreationParams) (db.CircuitCreation, error) {
	if req.InventoryRevision != nil {
		v, err := q.LockCircuitInventory(ctx, db.LockCircuitInventoryParams{WorkspaceID: params.WorkspaceID, ParentUserID: params.CreatorUserID, KitID: req.KitID})
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return db.CircuitCreation{}, err
		}
		if err != nil || !circuitInventoryMatches(v, req) {
			return db.CircuitCreation{}, errCircuitInventoryChanged
		}
	}
	return q.CreateCircuitCreation(ctx, params)
}
func (w *BuildWorker) circuitQuestion(ctx context.Context, job db.BuildJob, s db.BuildSession, p circuitBuildPlan, text string) error {
	question := buildstudio.ClarifyingQuestion{ID: fmt.Sprintf("q%d", s.Revision), Prompt: text, Choices: []buildstudio.QuestionChoice{}, Options: []string{}, AllowFreeText: true}
	if p.AwaitingMaterials {
		question.Choices = []buildstudio.QuestionChoice{{ID: "materials_ready", Label: localizedCircuit(p.Locale, "The parts box is ready", "元件盒准备好了")}}
		question.Options = []string{question.Choices[0].Label}
	}
	return w.finish(ctx, job, func(q *db.Queries) error {
		_, err := q.PauseBuildSession(ctx, db.PauseBuildSessionParams{ID: s.ID, Revision: s.Revision, Question: mustBuildJSON(question), Recipe: mustBuildJSON(p)})
		return err
	})
}
func (w *BuildWorker) finishReply(ctx context.Context, job db.BuildJob, s db.BuildSession, message string, unsupported bool) error {
	return w.finish(ctx, job, func(q *db.Queries) error {
		var value map[string]any
		_ = json.Unmarshal(s.Recipe, &value)
		if value == nil {
			value = map[string]any{}
		}
		value["summary"] = message
		recipe := mustBuildJSON(value)
		if unsupported {
			if _, err := q.SaveBuildSessionMessage(ctx, db.SaveBuildSessionMessageParams{ID: s.ID, Revision: s.Revision, Recipe: recipe}); err != nil {
				return err
			}
			return q.FailBuildSession(ctx, db.FailBuildSessionParams{ID: s.ID, Error: pgtype.Text{String: buildstudio.BuildErrorUnsupported, Valid: true}})
		}
		_, err := q.CompleteBuildConversationReply(ctx, db.CompleteBuildConversationReplyParams{ID: s.ID, Recipe: recipe})
		return err
	})
}
func (w *BuildWorker) processCircuit(ctx context.Context, job db.BuildJob, s db.BuildSession, answers map[string]string, history []map[string]string) error {
	var p circuitBuildPlan
	if json.Unmarshal(s.Recipe, &p) != nil {
		return w.failPermanently(ctx, job, buildstudio.BuildErrorRequirements, errors.New("invalid circuit plan"))
	}
	c, known := circuit.FindCatalog(p.KitID)
	if !known {
		return w.circuitQuestion(ctx, job, s, p, localizedCircuit(p.Locale, "Choose your electronic kit below, then tell me when it is ready.", "请先在下方选择你的电子套装，再告诉我准备好了。"))
	}
	if p.CatalogVersion != c.Version {
		return w.failPermanently(ctx, job, "CIRCUIT_CATALOG_CHANGED", errors.New("catalogue changed"))
	}
	if p.ProjectID == "" {
		if w.h.LLM == nil || !w.h.LLM.Enabled() {
			return w.finishReply(ctx, job, s, localizedCircuit(p.Locale, "Choose a reference project while AI is unavailable.", "AI 暂不可用，请选择一个参考项目。"), true)
		}
		planCtx, cancel := context.WithTimeout(ctx, buildIntentTimeout)
		d, err := circuit.PlanConversation(planCtx, w.h.LLM, c, map[string]any{"idea": s.Prompt, "answers": answers, "history": history, "previous_project_id": p.PreviousProjectID, "questions_remaining": max(0, maxBuildClarifications-int(s.Revision)+1)})
		cancel()
		if err != nil {
			return w.retry(ctx, job, err)
		}
		switch d.Outcome {
		case "clarify":
			if s.Revision > maxBuildClarifications {
				return w.finishReply(ctx, job, s, localizedCircuit(p.Locale, "Choose a reference project to continue.", "请先选择一个参考项目，再继续创作。"), true)
			}
			return w.circuitQuestion(ctx, job, s, p, d.Question)
		case "unsupported":
			return w.finishReply(ctx, job, s, d.Message, true)
		case "reply":
			project, _ := c.Project(d.ProjectID)
			return w.finishReply(ctx, job, s, localizedCircuit(p.Locale, project.Explanation.EN, project.Explanation.ZH), false)
		case "ready":
			p.ProjectID, p.Title = d.ProjectID, d.Title
		}
	}
	project, known := c.Project(p.ProjectID)
	if !known {
		return w.failPermanently(ctx, job, buildstudio.BuildErrorUnsupported, errors.New("unknown project"))
	}
	if p.Title == "" {
		p.Title = localizedCircuit(p.Locale, project.Title.EN, project.Title.ZH)
	}
	inventory, err := w.h.Queries.GetCircuitInventory(ctx, db.GetCircuitInventoryParams{WorkspaceID: s.WorkspaceID, ParentUserID: s.CreatorUserID, KitID: p.KitID})
	if errors.Is(err, pgx.ErrNoRows) || p.InventoryRevision <= 0 {
		p.AwaitingMaterials = true
		return w.circuitQuestion(ctx, job, s, p, localizedCircuit(p.Locale, "Ask a parent to confirm the component box below, then tell me when it is ready.", "请家长在下方核对并保存元件盒，再告诉我准备好了。"))
	}
	if err != nil {
		return w.retry(ctx, job, err)
	}
	req := circuitCreateRequest{KitID: p.KitID, CatalogVersion: p.CatalogVersion, InventoryRevision: &p.InventoryRevision, ProjectID: p.ProjectID, Prompt: s.Prompt, Locale: p.Locale}
	if json.Unmarshal(inventory.Quantities, &req.Inventory) != nil || !circuitInventoryMatches(inventory, req) {
		return w.failPermanently(ctx, job, "CIRCUIT_INVENTORY_CHANGED", errCircuitInventoryChanged)
	}
	doc, err := compileCircuitDocument(c, req, p.Title, "conversation-v1")
	if err != nil {
		return w.failPermanently(ctx, job, "CIRCUIT_VALIDATION_FAILED", err)
	}
	p.Summary = localizedCircuit(p.Locale, project.Description.EN, project.Description.ZH)
	err = w.finish(ctx, job, func(q *db.Queries) error {
		creation, err := saveCircuitDocument(ctx, q, req, db.CreateCircuitCreationParams{WorkspaceID: s.WorkspaceID, CreatorUserID: s.CreatorUserID, ChildProfileID: s.ChildProfileID, ActorKey: circuitActorKey(s.ChildProfileID), ClientRequestID: s.ID, RequestHash: s.RequestHash, Document: mustBuildJSON(doc)})
		if err != nil {
			return err
		}
		_, err = q.CompleteBuildConversationReply(ctx, db.CompleteBuildConversationReplyParams{ID: s.ID, Recipe: mustBuildJSON(p), CircuitCreationID: creation.ID})
		return err
	})
	if errors.Is(err, errCircuitInventoryChanged) {
		return w.failPermanently(ctx, job, "CIRCUIT_INVENTORY_CHANGED", err)
	}
	return err
}
func localizedCircuit(locale, en, zh string) string {
	if strings.HasPrefix(locale, "zh") {
		return zh
	}
	return en
}
