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
	Prompt              string                  `json:"prompt"`
	ClientRequestID     string                  `json:"client_request_id"`
	Design              *buildstudio.DesignSpec `json:"design,omitempty"`
	SourceCreationID    string                  `json:"source_creation_id,omitempty"`
	ExpectedContentHash string                  `json:"expected_content_hash,omitempty"`
}

type submitBuildAnswersRequest struct {
	Answers  map[string]string `json:"answers"`
	Revision int32             `json:"revision,omitempty"`
}

type buildSessionResponse struct {
	ID         string                          `json:"id"`
	Revision   int32                           `json:"revision"`
	Phase      string                          `json:"phase"`
	Summary    string                          `json:"summary,omitempty"`
	Message    string                          `json:"message,omitempty"`
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
	var req createBuildSessionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 48<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Design == nil && (h.LLM == nil || !h.LLM.Enabled()) {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "Build Studio requires an LLM configuration", "code": "build_unavailable", "reason": "llm_not_configured"})
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
	status := "queued"
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
	var recipe *buildstudio.AssemblyRecipe
	if req.SourceCreationID != "" {
		sourceID, valid := parseUUIDOrBadRequest(w, req.SourceCreationID, "source_creation_id")
		if !valid {
			return
		}
		source, loadErr := qtx.GetBuildCreationInWorkspace(r.Context(), db.GetBuildCreationInWorkspaceParams{ID: sourceID, WorkspaceID: workspaceID, CreatorUserID: userID, ChildProfileID: childProfileID})
		if errors.Is(loadErr, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "source creation not found")
			return
		}
		if loadErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to load source creation")
			return
		}
		previous, decodeErr := toBuildCreationResponse(source)
		if decodeErr != nil {
			writeError(w, http.StatusInternalServerError, "invalid source creation")
			return
		}
		if req.ExpectedContentHash == "" || req.ExpectedContentHash != previous.BuildPlan.ContentHash {
			writeError(w, http.StatusConflict, "source design does not match expected content")
			return
		}
		recipe = &previous.Recipe
		if recipe.Metadata == nil {
			recipe.Metadata = map[string]string{}
		}
		recipe.Metadata["parent_creation_id"], recipe.Metadata["parent_hash"] = uuidToString(source.ID), previous.BuildPlan.ContentHash
	}
	if req.Design != nil {
		if _, designErr := buildstudio.RasterizeDesign(*req.Design); designErr != nil {
			writeError(w, http.StatusBadRequest, "invalid shape design")
			return
		}
		if recipe == nil {
			recipe = &buildstudio.AssemblyRecipe{Title: string([]rune(req.Prompt)[:min(24, len([]rune(req.Prompt)))]), Subject: "custom", Archetype: "custom", Metadata: map[string]string{}, Palette: []int{}, Features: []string{}, Requirements: []string{}}
		}
		recipe.Version, recipe.Design, recipe.Modules = 3, req.Design, []buildstudio.ModuleInstance{}
		recipe.Constraints.RequiredModules = []string{}
		recipe.Constraints.ExactColors = true
		recipe.Prompt = req.Prompt
		recipe.Summary = string([]rune(req.Prompt)[:min(240, len([]rune(req.Prompt)))])
		recipe.Metadata["module_library_version"] = buildstudio.ModuleLibraryVersion
		recipe.Metadata["shape_generator_version"] = buildstudio.ShapeGeneratorVersion
	}
	var recipeJSON []byte
	if recipe != nil {
		recipeJSON = mustBuildJSON(recipe)
	}

	session, err := qtx.CreateBuildSession(r.Context(), db.CreateBuildSessionParams{
		WorkspaceID: workspaceID, CreatorUserID: userID, ChildProfileID: childProfileID,
		ClientRequestID: clientRequestID, Prompt: req.Prompt,
		Status: status, Question: nil, Answers: []byte(`{}`), InventorySnapshot: []byte(`{}`), Recipe: recipeJSON, DirectCompile: req.Design != nil,
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

	current, err := qtx.LockBuildSessionForAnswer(r.Context(), db.LockBuildSessionForAnswerParams{ID: sessionID, WorkspaceID: wsUUID, CreatorUserID: userID, ChildProfileID: childProfileID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "build session not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load build session")
		return
	}
	normalized, duplicate, err := validateBuildAnswer(current, req)
	if err != nil {
		writeError(w, http.StatusConflict, "answer does not match the current question")
		return
	}
	if duplicate {
		writeJSON(w, http.StatusOK, toBuildSessionResponse(current))
		return
	}
	answers, _ := json.Marshal(normalized)
	session, err := qtx.SubmitBuildSessionAnswers(r.Context(), db.SubmitBuildSessionAnswersParams{
		Answers: answers, Revision: current.Revision, ID: sessionID, WorkspaceID: wsUUID, CreatorUserID: userID, ChildProfileID: childProfileID,
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
	if r.URL.Query().Get("view") == "summary" {
		h.ListBuildCreationSummaries(w, r)
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
	response := buildSessionResponse{Revision: row.Revision, Phase: row.Phase, ID: uuidToString(row.ID), Prompt: row.Prompt, Status: row.Status, Answers: map[string]string{}, CreationID: uuidToString(row.CreationID)}
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
		switch row.Error.String {
		case buildstudio.BuildErrorSearchLimit, buildstudio.BuildErrorInsufficientInventory, buildstudio.BuildErrorCountUnsupported, buildstudio.BuildErrorStructureInvalid, buildstudio.BuildErrorUnsupported, buildstudio.BuildErrorRequirements, "BUILD_CANCELLED":
			response.Error = row.Error.String
		default:
			response.Error = "BUILD_GENERATION_FAILED"
		}
	}

	if len(row.Recipe) > 0 {
		var recipe buildstudio.AssemblyRecipe
		if json.Unmarshal(row.Recipe, &recipe) == nil {
			response.Summary = recipe.Summary
			if response.Error == buildstudio.BuildErrorUnsupported {
				response.Message = recipe.Summary
			}
		}
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
	// Creations written before the response contract was normalized contain
	// JSON null for successful validation issues and unlimited inventory items.
	// Canonicalize them at the API boundary so historical rows are immediately
	// readable without a destructive data migration.
	if response.Validation.Issues == nil {
		response.Validation.Issues = []buildstudio.ValidationIssue{}
	}
	if response.Validation.UsedParts == nil {
		response.Validation.UsedParts = map[string]int{}
	}
	if response.BuildPlan.Validation.Issues == nil {
		response.BuildPlan.Validation.Issues = []buildstudio.ValidationIssue{}
	}
	if response.BuildPlan.Validation.UsedParts == nil {
		response.BuildPlan.Validation.UsedParts = map[string]int{}
	}
	if response.BuildPlan.Inventory != nil && response.BuildPlan.Inventory.Items == nil {
		response.BuildPlan.Inventory.Items = []buildstudio.InventoryItem{}
	}
	return response, nil
}

func (h *Handler) CancelBuildSession(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	id, err := util.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid build session")
		return
	}
	var req struct {
		Revision int32 `json:"revision"`
	}
	if err = json.NewDecoder(http.MaxBytesReader(w, r.Body, 1024)).Decode(&req); err != nil || req.Revision < 1 {
		writeError(w, http.StatusBadRequest, "revision is required")
		return
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel build")
		return
	}
	defer tx.Rollback(r.Context())
	q := h.Queries.WithTx(tx)
	session, err := q.LockBuildSessionForAnswer(r.Context(), db.LockBuildSessionForAnswerParams{ID: id, WorkspaceID: ws, CreatorUserID: user, ChildProfileID: child})
	if err != nil {
		writeError(w, http.StatusNotFound, "build session not found")
		return
	}
	if session.Revision != req.Revision {
		writeError(w, http.StatusConflict, "build session changed")
		return
	}
	if session.Status == "completed" || session.Status == "failed" {
		writeJSON(w, http.StatusOK, toBuildSessionResponse(session))
		return
	}
	if err = q.CancelBuildJob(r.Context(), id); err == nil {
		err = q.FailBuildSession(r.Context(), db.FailBuildSessionParams{ID: id, Error: pgtype.Text{String: "BUILD_CANCELLED", Valid: true}})
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to cancel build")
		return
	}
	session.Status = "failed"
	session.Error = pgtype.Text{String: "BUILD_CANCELLED", Valid: true}
	writeJSON(w, http.StatusOK, toBuildSessionResponse(session))
}
