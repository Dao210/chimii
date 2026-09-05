package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chimii-ai/chimii/server/internal/circuit"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type circuitTrialResponse struct {
	ID            string `json:"id"`
	CreationID    string `json:"creation_id"`
	DocumentHash  string `json:"document_hash"`
	HardwareLabel string `json:"hardware_label"`
	Result        string `json:"result"`
	Notes         string `json:"notes"`
	CreatedAt     string `json:"created_at"`
	EvidenceKind  string `json:"evidence_kind"`
}

func circuitTrial(v db.CircuitTrial) circuitTrialResponse {
	return circuitTrialResponse{uuidToString(v.ID), uuidToString(v.CreationID), v.DocumentHash, v.HardwareLabel, v.Result, v.Notes, v.CreatedAt.Time.UTC().Format(time.RFC3339), "family_report"}
}

func (h *Handler) ListCircuitTrials(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	_, err := h.Queries.GetCircuitCreation(r.Context(), db.GetCircuitCreationParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "circuit not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not load circuit")
		return
	}
	vs, err := h.Queries.ListCircuitTrials(r.Context(), db.ListCircuitTrialsParams{CreationID: id, WorkspaceID: ws, ParentUserID: user, ActorKey: circuitActorKey(child)})
	if err != nil {
		writeError(w, 500, "could not load trials")
		return
	}
	trials := []circuitTrialResponse{}
	for _, v := range vs {
		trials = append(trials, circuitTrial(v))
	}
	writeJSON(w, 200, map[string]any{"trials": trials})
}

func (h *Handler) CreateCircuitTrial(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	creationID, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	var req struct {
		ClientRequestID string `json:"client_request_id"`
		HardwareLabel   string `json:"hardware_label"`
		Result          string `json:"result"`
		Notes           string `json:"notes"`
		AdultChecked    bool   `json:"adult_checked"`
	}
	if !decodeCircuitRequest(w, r, &req) {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, req.ClientRequestID, "client_request_id")
	if !ok {
		return
	}
	req.HardwareLabel = strings.TrimSpace(req.HardwareLabel)
	req.Notes = strings.TrimSpace(req.Notes)
	if !req.AdultChecked || len([]rune(req.HardwareLabel)) < 1 || len([]rune(req.HardwareLabel)) > 120 || len([]rune(req.Notes)) > 1000 || (req.Result != "worked" && req.Result != "needs_help") {
		writeError(w, 400, "check the hardware and provide a trial result")
		return
	}
	data, err := json.Marshal(req)
	if err != nil {
		writeError(w, 400, "invalid trial")
		return
	}
	sum := sha256.Sum256(data)
	hash := hex.EncodeToString(sum[:])
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save trial")
		return
	}
	defer tx.Rollback(r.Context())
	q := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), q, ws, user); err != nil {
		writeError(w, 404, "workspace membership not found")
		return
	}
	creation, err := q.GetCircuitCreation(r.Context(), db.GetCircuitCreationParams{ID: creationID, WorkspaceID: ws, CreatorUserID: user, ActorKey: circuitActorKey(child)})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "circuit not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not load circuit")
		return
	}
	var doc circuit.Document
	if err := json.Unmarshal(creation.Document, &doc); err != nil || len(doc.Project.Steps) == 0 {
		writeError(w, 500, "invalid saved circuit")
		return
	}
	lookup := db.GetCircuitTrialParams{ID: id, WorkspaceID: ws, ParentUserID: user, ActorKey: circuitActorKey(child)}
	replay := func(v db.CircuitTrial) {
		if v.RequestHash != hash || v.CreationID != creationID {
			writeJSON(w, 409, map[string]string{"error": "trial request already used", "code": "circuit_request_conflict"})
			return
		}
		writeJSON(w, 200, circuitTrial(v))
	}
	if old, err := q.GetCircuitTrial(r.Context(), lookup); err == nil {
		replay(old)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 500, "could not check trial")
		return
	}
	if int(creation.CurrentStep) != len(doc.Project.Steps)-1 {
		writeError(w, 400, "finish the assembly before recording a trial")
		return
	}
	v, err := q.CreateCircuitTrial(r.Context(), db.CreateCircuitTrialParams{ID: id, WorkspaceID: ws, ParentUserID: user, ActorKey: circuitActorKey(child), CreationID: creationID, DocumentHash: doc.ContentHash, RequestHash: hash, HardwareLabel: req.HardwareLabel, Result: req.Result, Notes: req.Notes})
	if errors.Is(err, pgx.ErrNoRows) {
		old, err := q.GetCircuitTrial(r.Context(), lookup)
		if err != nil {
			writeError(w, 500, "could not replay trial")
			return
		}
		replay(old)
		return
	}
	if err != nil {
		writeError(w, 500, "could not save trial")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not save trial")
		return
	}
	writeJSON(w, 201, circuitTrial(v))
}
