package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/middleware"
	"github.com/chimii-ai/chimii/server/internal/util"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxBuildPromptRunes     = 280
	maxActiveBuildsPerActor = 2
)

type createBuildSessionRequest struct {
	Prompt          string `json:"prompt"`
	ClientRequestID string `json:"client_request_id"`
}

type submitBuildAnswersRequest struct {
	Answers map[string]string `json:"answers"`
}

type buildSessionResponse struct {
	ID         string                          `json:"id"`
	Prompt     string                          `json:"prompt"`
	Status     string                          `json:"status"`
	Question   *buildstudio.ClarifyingQuestion `json:"question,omitempty"`
	Answers    map[string]string               `json:"answers"`
	CreationID string                          `json:"creation_id,omitempty"`
	Error      string                          `json:"error,omitempty"`
	CreatedAt  string                          `json:"created_at"`
	UpdatedAt  string                          `json:"updated_at"`
}

type buildCreationResponse struct {
	ID         string                       `json:"id"`
	SessionID  string                       `json:"session_id"`
	Title      string                       `json:"title"`
	Prompt     string                       `json:"prompt"`
	Archetype  string                       `json:"archetype"`
	Recipe     buildstudio.AssemblyRecipe   `json:"recipe"`
	BuildPlan  buildstudio.BuildPlan        `json:"build_plan"`
	Validation buildstudio.ValidationReport `json:"validation"`
	CreatedAt  string                       `json:"created_at"`
}

func (h *Handler) CreateBuildSession(w http.ResponseWriter, r *http.Request) {
	if h.LLM == nil || !h.LLM.Enabled() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Build Studio requires an LLM configuration", "code": "build_unavailable", "reason": "llm_not_configured"})
		return
	}
	var req createBuildSessionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Prompt = strings.TrimSpace(req.Prompt)
	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	if len([]rune(req.Prompt)) > maxBuildPromptRunes {
		writeError(w, http.StatusBadRequest, "prompt is too long")
		return
	}
	workspaceID, userID, childProfileID, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	clientRequestID, err := util.ParseUUID(req.ClientRequestID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "client_request_id must be a UUID")
		return
	}
	question := buildstudio.QuestionFor(req.Prompt)
	status := "queued"
	var questionJSON []byte
	if question != nil {
		status = "clarifying"
		questionJSON, _ = json.Marshal(question)
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start build")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), qtx, workspaceID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace membership not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to start build")
		return
	}
	actorKey := "build:" + uuidToString(workspaceID) + ":" + uuidToString(userID) + ":parent"
	if childProfileID.Valid {
		actorKey = "build:" + uuidToString(workspaceID) + ":" + uuidToString(userID) + ":child:" + uuidToString(childProfileID)
	}
	if err := qtx.LockBuildActorCapacity(r.Context(), actorKey); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check build capacity")
		return
	}
	requestScope := db.GetBuildSessionByClientRequestParams{
		WorkspaceID: workspaceID, CreatorUserID: userID, ClientRequestID: clientRequestID, ChildProfileID: childProfileID,
	}
	if existing, lookupErr := qtx.GetBuildSessionByClientRequest(r.Context(), requestScope); lookupErr == nil {
		writeJSON(w, http.StatusAccepted, toBuildSessionResponse(existing))
		return
	} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to check build request")
		return
	}
	activeCount, err := qtx.CountActiveBuildSessions(r.Context(), db.CountActiveBuildSessionsParams{
		WorkspaceID: workspaceID, CreatorUserID: userID, ChildProfileID: childProfileID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to check build capacity")
		return
	}
	if activeCount >= maxActiveBuildsPerActor {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{
			"error": "finish an active build before starting another", "code": "BUILD_ACTIVE_LIMIT",
		})
		return
	}

	session, err := qtx.CreateBuildSession(r.Context(), db.CreateBuildSessionParams{
		WorkspaceID: workspaceID, CreatorUserID: userID, ChildProfileID: childProfileID,
		ClientRequestID: clientRequestID, Prompt: req.Prompt,
		Status: status, Question: questionJSON, Answers: []byte(`{}`),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start build")
		return
	}
	if status == "queued" {
		if _, err := qtx.EnqueueBuildJob(r.Context(), db.EnqueueBuildJobParams{WorkspaceID: workspaceID, SessionID: session.ID}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to queue build")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start build")
		return
	}
	if status == "queued" {
		h.BuildWorker.Notify()
	}
	writeJSON(w, http.StatusAccepted, toBuildSessionResponse(session))
}

func (h *Handler) SubmitBuildAnswers(w http.ResponseWriter, r *http.Request) {
	wsUUID, userID, childProfileID, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	sessionID, err := util.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid build session")
		return
	}
	var req submitBuildAnswersRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil || len(req.Answers) == 0 {
		writeError(w, http.StatusBadRequest, "an answer is required")
		return
	}
	for key, value := range req.Answers {
		value = strings.TrimSpace(value)
		if key == "" || value == "" || len([]rune(value)) > 120 {
			writeError(w, http.StatusBadRequest, "invalid answer")
			return
		}
		req.Answers[key] = value
	}
	answers, _ := json.Marshal(req.Answers)
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save answer")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), qtx, wsUUID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace membership not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to save answer")
		return
	}
	session, err := qtx.SubmitBuildSessionAnswers(r.Context(), db.SubmitBuildSessionAnswersParams{
		Answers: answers, ID: sessionID, WorkspaceID: wsUUID, CreatorUserID: userID, ChildProfileID: childProfileID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusConflict, "build session is not waiting for an answer")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save answer")
		return
	}
	if _, err := qtx.EnqueueBuildJob(r.Context(), db.EnqueueBuildJobParams{WorkspaceID: wsUUID, SessionID: session.ID}); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to queue build")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save answer")
		return
	}
	h.BuildWorker.Notify()
	writeJSON(w, http.StatusOK, toBuildSessionResponse(session))
}

func (h *Handler) GetBuildSession(w http.ResponseWriter, r *http.Request) {
	wsUUID, userID, childProfileID, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	id, err := util.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid build session")
		return
	}
	session, err := h.Queries.GetBuildSessionInWorkspace(r.Context(), db.GetBuildSessionInWorkspaceParams{
		ID: id, WorkspaceID: wsUUID, CreatorUserID: userID, ChildProfileID: childProfileID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "build session not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load build session")
		return
	}
	writeJSON(w, http.StatusOK, toBuildSessionResponse(session))
}

func (h *Handler) ListBuildCreations(w http.ResponseWriter, r *http.Request) {
	wsUUID, userID, childProfileID, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	pageSize := int32(60)
	if value, parseErr := strconv.Atoi(r.URL.Query().Get("limit")); parseErr == nil && value > 0 && value <= 100 {
		pageSize = int32(value)
	}
	rows, err := h.Queries.ListBuildCreations(r.Context(), db.ListBuildCreationsParams{
		WorkspaceID: wsUUID, CreatorUserID: userID, ChildProfileID: childProfileID, PageSize: pageSize, PageOffset: 0,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load creations")
		return
	}
	items := make([]buildCreationResponse, 0, len(rows))
	for _, row := range rows {
		item, err := toBuildCreationResponse(row)
		if err == nil {
			items = append(items, item)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"creations": items})
}

func (h *Handler) GetBuildCreation(w http.ResponseWriter, r *http.Request) {
	row, ok := h.loadBuildCreation(w, r)
	if !ok {
		return
	}
	response, err := toBuildCreationResponse(row)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "creation data is invalid")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) ExportBuildCreationMPD(w http.ResponseWriter, r *http.Request) {
	row, ok := h.loadBuildCreation(w, r)
	if !ok {
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="chimii-creation.mpd"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(row.LdrawMpd))
}

func (h *Handler) loadBuildCreation(w http.ResponseWriter, r *http.Request) (db.BuildCreation, bool) {
	wsUUID, userID, childProfileID, ok := buildActorScope(w, r)
	if !ok {
		return db.BuildCreation{}, false
	}
	id, err := util.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid creation")
		return db.BuildCreation{}, false
	}
	row, err := h.Queries.GetBuildCreationInWorkspace(r.Context(), db.GetBuildCreationInWorkspaceParams{
		ID: id, WorkspaceID: wsUUID, CreatorUserID: userID, ChildProfileID: childProfileID,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "creation not found")
		return db.BuildCreation{}, false
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load creation")
		return db.BuildCreation{}, false
	}
	return row, true
}

func buildActorScope(w http.ResponseWriter, r *http.Request) (workspaceID, userID, childProfileID pgtype.UUID, ok bool) {
	ws, err := util.ParseUUID(middleware.WorkspaceIDFromContext(r.Context()))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid workspace")
		return workspaceID, userID, childProfileID, false
	}
	user, err := util.ParseUUID(r.Header.Get("X-User-ID"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "user not authenticated")
		return workspaceID, userID, childProfileID, false
	}
	if r.Header.Get("X-Actor-Source") == "child_session" {
		profile, err := util.ParseUUID(r.Header.Get("X-Child-Profile-ID"))
		if err != nil {
			writeError(w, http.StatusUnauthorized, "child profile is invalid")
			return workspaceID, userID, childProfileID, false
		}
		childProfileID = profile
	}
	return ws, user, childProfileID, true
}

// buildActorIDs is shared by child-mode handlers that only need the bound
// workspace and parent identity. Keeping it as a narrow wrapper avoids
// duplicating the authoritative actor parsing rules.
func buildActorIDs(w http.ResponseWriter, r *http.Request) (workspaceID, userID pgtype.UUID, ok bool) {
	workspaceID, userID, _, ok = buildActorScope(w, r)
	return workspaceID, userID, ok
}

// lockWorkspaceMemberForScopedWrite is the writer half of the explicit
// cleanup protocol for Build Studio and child-mode tables, which intentionally
// have no foreign keys. Lock order is always workspace then member.
func lockWorkspaceMemberForScopedWrite(ctx context.Context, queries *db.Queries, workspaceID, userID pgtype.UUID) error {
	if _, err := queries.LockWorkspaceForScopedWrite(ctx, workspaceID); err != nil {
		return err
	}
	_, err := queries.LockWorkspaceMemberForScopedWrite(ctx, db.LockWorkspaceMemberForScopedWriteParams{
		WorkspaceID: workspaceID,
		UserID:      userID,
	})
	return err
}

func toBuildSessionResponse(row db.BuildSession) buildSessionResponse {
	response := buildSessionResponse{ID: uuidToString(row.ID), Prompt: row.Prompt, Status: row.Status, Answers: map[string]string{}, CreationID: uuidToString(row.CreationID)}
	if row.CreatedAt.Valid {
		response.CreatedAt = row.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if row.UpdatedAt.Valid {
		response.UpdatedAt = row.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if row.Error.Valid {
		// The database retains an operator-facing cause for diagnosis. Child
		// responses expose only a stable product code and never raw LLM/upstream
		// text, URLs, credentials, or stack details.
		response.Error = "BUILD_GENERATION_FAILED"
	}
	if len(row.Question) > 0 {
		var q buildstudio.ClarifyingQuestion
		if json.Unmarshal(row.Question, &q) == nil {
			response.Question = &q
		}
	}
	if len(row.Answers) > 0 {
		_ = json.Unmarshal(row.Answers, &response.Answers)
	}
	return response
}

func toBuildCreationResponse(row db.BuildCreation) (buildCreationResponse, error) {
	response := buildCreationResponse{ID: uuidToString(row.ID), SessionID: uuidToString(row.SessionID), Title: row.Title, Prompt: row.Prompt, Archetype: row.Archetype}
	if row.CreatedAt.Valid {
		response.CreatedAt = row.CreatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	if err := json.Unmarshal(row.Recipe, &response.Recipe); err != nil {
		return response, err
	}
	if err := json.Unmarshal(row.BuildPlan, &response.BuildPlan); err != nil {
		return response, err
	}
	if err := json.Unmarshal(row.Validation, &response.Validation); err != nil {
		return response, err
	}
	return response, nil
}
