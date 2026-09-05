package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type buildProgressResponse struct {
	ID          string  `json:"id"`
	CurrentStep int32   `json:"current_step"`
	CompletedAt *string `json:"completed_at"`
	Revision    int32   `json:"revision"`
	StepCount   int     `json:"step_count"`
}

func buildProgress(id pgtype.UUID, step, revision int32, completed pgtype.Timestamptz, validation []byte) buildProgressResponse {
	var v buildstudio.ValidationReport
	_ = json.Unmarshal(validation, &v)
	return buildProgressResponse{uuidToString(id), step, timestampToPtr(completed), revision, v.StepCount}
}

func (h *Handler) GetBuildProgress(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	id, ok := parseUUIDOrBadRequest(w, chi.URLParam(r, "id"), "id")
	if !ok {
		return
	}
	v, err := h.Queries.GetBuildProgress(r.Context(), db.GetBuildProgressParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "creation not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not read progress")
		return
	}
	writeJSON(w, 200, buildProgress(v.ID, v.CurrentStep, v.ProgressRevision, v.CompletedAt, v.Validation))
}

func (h *Handler) UpdateBuildProgress(w http.ResponseWriter, r *http.Request) {
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
		Completed        bool   `json:"completed"`
		ExpectedRevision *int32 `json:"expected_revision"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil || req.CurrentStep < 1 || req.ExpectedRevision == nil || *req.ExpectedRevision < 0 {
		writeError(w, 400, "valid step and expected_revision are required")
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
	existing, err := q.GetBuildProgress(r.Context(), db.GetBuildProgressParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, 404, "creation not found")
		return
	}
	if err != nil {
		writeError(w, 500, "could not read progress")
		return
	}
	current := buildProgress(existing.ID, existing.CurrentStep, existing.ProgressRevision, existing.CompletedAt, existing.Validation)
	if int(req.CurrentStep) > current.StepCount || (req.Completed && int(req.CurrentStep) != current.StepCount) {
		writeError(w, 400, "step is outside the build plan")
		return
	}
	v, err := q.UpdateBuildProgress(r.Context(), db.UpdateBuildProgressParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child, CurrentStep: req.CurrentStep, Completed: req.Completed, ExpectedRevision: *req.ExpectedRevision})
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, 409, map[string]string{"code": "BUILD_PROGRESS_CONFLICT", "error": "Progress changed; reload and try again"})
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
	writeJSON(w, 200, buildProgress(v.ID, v.CurrentStep, v.ProgressRevision, v.CompletedAt, v.Validation))
}

func (h *Handler) ListBuildCreationSummaries(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	rows, err := h.Queries.ListBuildCreationSummaries(r.Context(), db.ListBuildCreationSummariesParams{WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child, PageSize: 60})
	if err != nil {
		writeError(w, 500, "could not load creations")
		return
	}
	type summary struct {
		ID        string                `json:"id"`
		Title     string                `json:"title"`
		Prompt    string                `json:"prompt"`
		Archetype string                `json:"archetype"`
		PartCount int                   `json:"part_count"`
		StepCount int                   `json:"step_count"`
		Progress  buildProgressResponse `json:"progress"`
		CreatedAt string                `json:"created_at"`
	}
	items := make([]summary, 0, len(rows))
	for _, row := range rows {
		var v buildstudio.ValidationReport
		if err := json.Unmarshal(row.Validation, &v); err != nil {
			writeError(w, 500, "invalid creation summary")
			return
		}
		items = append(items, summary{uuidToString(row.ID), row.Title, row.Prompt, row.Archetype, v.PartCount, v.StepCount, buildProgress(row.ID, row.CurrentStep, row.ProgressRevision, row.CompletedAt, row.Validation), timestampToString(row.CreatedAt)})
	}
	writeJSON(w, 200, map[string]any{"creations": items})
}
