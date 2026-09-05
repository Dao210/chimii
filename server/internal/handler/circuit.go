package handler

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/chimii-ai/chimii/server/internal/circuit"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type circuitCreateRequest struct {
	ClientRequestID string         `json:"client_request_id"`
	KitID           string         `json:"kit_id"`
	CatalogVersion  string         `json:"catalog_version"`
	ProjectID       string         `json:"project_id"`
	Prompt          string         `json:"prompt"`
	Locale          string         `json:"locale"`
	Inventory       map[string]int `json:"inventory"`
}

type circuitCreationResponse struct {
	ID               string          `json:"id"`
	Document         json.RawMessage `json:"document"`
	CurrentStep      int32           `json:"current_step"`
	Observation      string          `json:"observation"`
	ProgressRevision int32           `json:"progress_revision"`
	CreatedAt        string          `json:"created_at"`
}

func circuitResponse(v db.CircuitCreation) circuitCreationResponse {
	return circuitCreationResponse{uuidToString(v.ID), v.Document, v.CurrentStep, v.Observation, v.ProgressRevision, v.CreatedAt.Time.UTC().Format(time.RFC3339)}
}

func circuitActorKey(child pgtype.UUID) string {
	if child.Valid {
		return uuidToString(child)
	}
	return "parent"
}

func (h *Handler) GetCircuitCatalog(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"catalog": circuit.StarterCatalog(), "ai_available": h.LLM != nil && h.LLM.Enabled()})
}

func decodeCircuitRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	if err := d.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func (h *Handler) CreateCircuitCreation(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	var req circuitCreateRequest
	if !decodeCircuitRequest(w, r, &req) {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, req.ClientRequestID, "client_request_id")
	if !ok {
		return
	}
	c := circuit.StarterCatalog()
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.KitID != c.KitID || req.CatalogVersion != c.Version {
		writeJSON(w, 409, map[string]string{"error": "Refresh the component catalogue", "code": "circuit_catalog_changed"})
		return
	}
	if req.Inventory == nil || len(req.Inventory) > len(c.Parts) || len([]rune(req.Prompt)) > 280 || ((req.ProjectID == "") == (req.Prompt == "")) {
		writeError(w, 400, "provide either a project or an idea, and an inventory")
		return
	}
	for part, n := range req.Inventory {
		p, exists := c.Part(part)
		if !exists || n < 0 || n > p.Quantity {
			writeError(w, 400, "invalid inventory")
			return
		}
	}
	requestBytes, err := json.Marshal(req)
	if err != nil {
		writeError(w, 400, "invalid request")
		return
	}
	sum := sha256.Sum256(requestBytes)
	hash := hex.EncodeToString(sum[:])
	lookup := db.GetCircuitCreationByRequestParams{ClientRequestID: id, WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child)}
	if existing, err := h.Queries.GetCircuitCreationByRequest(r.Context(), lookup); err == nil {
		h.writeCircuitReplay(w, existing, hash)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "could not check request")
		return
	}
	planner := "project"
	title := ""
	if req.ProjectID == "" {
		if h.LLM == nil || !h.LLM.Enabled() {
			writeJSON(w, 503, map[string]string{"error": "AI is unavailable; choose a project", "code": "circuit_ai_unavailable"})
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()
		intent, err := circuit.Plan(ctx, h.LLM, req.Prompt)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "Could not understand this idea; retry or choose a project", "code": "circuit_planning_failed"})
			return
		}
		if intent.ProjectID == "unsupported" {
			writeJSON(w, 422, map[string]string{"error": "This idea needs features outside the three supported projects", "code": "circuit_unsupported"})
			return
		}
		req.ProjectID = intent.ProjectID
		title = intent.Title
		planner = "llm-intent-v1"
	} else {
		p, exists := c.Project(req.ProjectID)
		if !exists {
			writeError(w, 400, "unknown circuit project")
			return
		}
		title = p.Title.EN
		if strings.HasPrefix(req.Locale, "zh") {
			title = p.Title.ZH
		}
	}
	doc, err := circuit.Compile(c, req.ProjectID, req.Prompt, title, planner, req.Inventory)
	if err != nil {
		writeJSON(w, 422, map[string]any{"error": "Circuit could not pass the reference checks", "code": "circuit_validation_failed", "validation": doc.Validation})
		return
	}
	data, err := json.Marshal(doc)
	if err != nil {
		writeError(w, 500, "could not encode circuit")
		return
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save circuit")
		return
	}
	defer tx.Rollback(r.Context())
	q := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), q, ws, user); err != nil {
		writeError(w, 404, "workspace membership not found")
		return
	}
	created, err := q.CreateCircuitCreation(r.Context(), db.CreateCircuitCreationParams{WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child, ActorKey: lookup.ActorKey, ClientRequestID: id, RequestHash: hash, Document: data})
	if errors.Is(err, pgx.ErrNoRows) {
		existing, readErr := q.GetCircuitCreationByRequest(r.Context(), lookup)
		if readErr != nil {
			writeError(w, 500, "could not replay request")
			return
		}
		h.writeCircuitReplay(w, existing, hash)
		return
	}
	if err != nil {
		writeError(w, 500, "could not save circuit")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not save circuit")
		return
	}
	writeJSON(w, http.StatusCreated, circuitResponse(created))
}

func (h *Handler) writeCircuitReplay(w http.ResponseWriter, v db.CircuitCreation, hash string) {
	if v.RequestHash != hash {
		writeJSON(w, 409, map[string]string{"error": "Request ID already used for a different circuit", "code": "circuit_request_conflict"})
		return
	}
	writeJSON(w, 200, circuitResponse(v))
}

func (h *Handler) GetCircuitCreation(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	v, err := h.Queries.GetCircuitCreation(r.Context(), db.GetCircuitCreationParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "circuit not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not load circuit")
		return
	}
	writeJSON(w, 200, circuitResponse(v))
}

func (h *Handler) ListCircuitCreations(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	vs, err := h.Queries.ListCircuitCreations(r.Context(), db.ListCircuitCreationsParams{WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child)})
	if err != nil {
		writeError(w, 500, "could not list circuits")
		return
	}
	writeJSON(w, 200, map[string]any{"creations": vs})
}

func (h *Handler) UpdateCircuitProgress(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	var req struct {
		CurrentStep      int32  `json:"current_step"`
		Observation      string `json:"observation"`
		ExpectedRevision int32  `json:"expected_revision"`
	}
	if !decodeCircuitRequest(w, r, &req) {
		return
	}
	if req.CurrentStep < 0 || req.ExpectedRevision < 0 || (req.Observation != "not_tried" && req.Observation != "worked" && req.Observation != "needs_help") {
		writeError(w, 400, "invalid progress")
		return
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save progress")
		return
	}
	defer tx.Rollback(r.Context())
	q := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), q, ws, user); err != nil {
		writeError(w, 404, "workspace membership not found")
		return
	}
	existing, err := q.GetCircuitCreation(r.Context(), db.GetCircuitCreationParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "circuit not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not load circuit")
		return
	}
	var doc circuit.Document
	if err := json.Unmarshal(existing.Document, &doc); err != nil {
		writeError(w, 500, "invalid saved circuit")
		return
	}
	if int(req.CurrentStep) >= len(doc.Project.Steps) || (req.Observation != "not_tried" && int(req.CurrentStep) != len(doc.Project.Steps)-1) {
		writeError(w, 400, "finish the assembly steps before recording an observation")
		return
	}
	v, err := q.UpdateCircuitProgress(r.Context(), db.UpdateCircuitProgressParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child), CurrentStep: req.CurrentStep, Observation: req.Observation, ExpectedRevision: req.ExpectedRevision})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 409, map[string]string{"error": "Progress changed; refresh and try again", "code": "circuit_progress_conflict"})
		return
	}
	if err != nil {
		writeError(w, 500, "could not save progress")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not save progress")
		return
	}
	writeJSON(w, 200, circuitResponse(v))
}
