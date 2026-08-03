package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/chimii-ai/chimii/server/internal/auth"
	"github.com/chimii-ai/chimii/server/internal/util"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const childSessionTTL = 12 * time.Hour

type childProfileResponse struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	AvatarSeed  string `json:"avatar_seed"`
}

type childModeResponse struct {
	Mode         string                `json:"mode"`
	Profile      *childProfileResponse `json:"profile,omitempty"`
	Capabilities []string              `json:"capabilities"`
}

type createChildProfileRequest struct {
	DisplayName string `json:"display_name"`
	AvatarSeed  string `json:"avatar_seed"`
	PIN         string `json:"pin"`
}

func (h *Handler) GetChildMode(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Actor-Source") != "child_session" {
		writeJSON(w, http.StatusOK, childModeResponse{Mode: "parent", Capabilities: []string{"*"}})
		return
	}
	writeJSON(w, http.StatusOK, childModeResponse{
		Mode:         "child",
		Profile:      &childProfileResponse{ID: r.Header.Get("X-Child-Profile-ID"), DisplayName: r.Header.Get("X-Child-Display-Name")},
		Capabilities: []string{"build:create", "creations:read", "creations:export"},
	})
}

func (h *Handler) ListChildProfiles(w http.ResponseWriter, r *http.Request) {
	workspaceID, userID, ok := buildActorIDs(w, r)
	if !ok {
		return
	}
	profiles, err := h.Queries.ListChildProfiles(r.Context(), db.ListChildProfilesParams{WorkspaceID: workspaceID, ParentUserID: userID})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list child profiles")
		return
	}
	result := make([]childProfileResponse, 0, len(profiles))
	for _, profile := range profiles {
		result = append(result, childProfileToResponse(profile))
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": result})
}

func (h *Handler) CreateChildProfile(w http.ResponseWriter, r *http.Request) {
	workspaceID, userID, ok := buildActorIDs(w, r)
	if !ok {
		return
	}
	var req createChildProfileRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if len([]rune(req.DisplayName)) < 1 || len([]rune(req.DisplayName)) > 24 {
		writeError(w, http.StatusBadRequest, "display name must be 1-24 characters")
		return
	}
	pinHash, err := auth.HashChildPIN(req.PIN)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create child profile")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), qtx, workspaceID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace membership not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create child profile")
		return
	}
	profile, err := qtx.CreateChildProfile(r.Context(), db.CreateChildProfileParams{
		WorkspaceID: workspaceID, ParentUserID: userID, DisplayName: req.DisplayName,
		AvatarSeed: strings.TrimSpace(req.AvatarSeed), PinHash: pinHash,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create child profile")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create child profile")
		return
	}
	writeJSON(w, http.StatusCreated, childProfileToResponse(profile))
}

func (h *Handler) EnterChildMode(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Actor-Source") == "child_session" {
		writeError(w, http.StatusConflict, "already in child mode")
		return
	}
	workspaceID, userID, ok := buildActorIDs(w, r)
	if !ok {
		return
	}
	profileID, err := util.ParseUUID(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid child profile")
		return
	}
	token, err := generateChildSessionToken()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start child mode")
		return
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start child mode")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), qtx, workspaceID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace membership not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to start child mode")
		return
	}
	profile, err := qtx.GetChildProfileForParent(r.Context(), db.GetChildProfileForParentParams{ID: profileID, WorkspaceID: workspaceID, ParentUserID: userID})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusNotFound, "child profile not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load child profile")
		return
	}
	session, err := qtx.CreateChildSession(r.Context(), db.CreateChildSessionParams{
		TokenHash: auth.HashToken(token), ProfileID: profile.ID, WorkspaceID: workspaceID,
		ParentUserID: userID, ExpiresAt: pgtype.Timestamptz{Time: time.Now().Add(childSessionTTL), Valid: true},
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start child mode")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to start child mode")
		return
	}
	if err := auth.SetAuthCookies(w, token); err != nil {
		_ = h.Queries.RevokeChildSession(r.Context(), session.ID)
		writeError(w, http.StatusInternalServerError, "failed to start child mode")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"token": token,
		"mode":  childModeResponse{Mode: "child", Profile: ptrChildProfile(childProfileToResponse(profile)), Capabilities: []string{"build:create", "creations:read", "creations:export"}},
	})
}

func (h *Handler) ExitChildMode(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("X-Actor-Source") != "child_session" {
		writeError(w, http.StatusConflict, "not in child mode")
		return
	}
	var req struct {
		PIN string `json:"pin"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "PIN is required")
		return
	}
	sessionID, err := util.ParseUUID(r.Header.Get("X-Child-Session-ID"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid child session")
		return
	}
	session, err := h.Queries.GetActiveChildSessionByID(r.Context(), sessionID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid child session")
		return
	}
	if session.UnlockLockedUntil.Valid && session.UnlockLockedUntil.Time.After(time.Now()) {
		writeJSON(w, http.StatusTooManyRequests, map[string]any{"error": "too many PIN attempts", "retry_at": session.UnlockLockedUntil.Time.UTC().Format(time.RFC3339)})
		return
	}
	if !auth.VerifyChildPIN(session.PinHash, req.PIN) {
		_, _ = h.Queries.RecordChildUnlockFailure(r.Context(), session.ID)
		writeError(w, http.StatusUnauthorized, "incorrect parent PIN")
		return
	}
	_ = h.Queries.ResetChildUnlockFailures(r.Context(), session.ID)
	user, err := h.Queries.GetUser(r.Context(), session.ParentUserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore parent mode")
		return
	}
	parentToken, err := h.issueJWT(user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore parent mode")
		return
	}
	if err := auth.SetAuthCookies(w, parentToken); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to restore parent mode")
		return
	}
	_ = h.Queries.RevokeChildSession(r.Context(), session.ID)
	writeJSON(w, http.StatusOK, LoginResponse{Token: parentToken, User: userToResponse(user)})
}

func generateChildSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "mch_" + hex.EncodeToString(bytes), nil
}

func childProfileToResponse(profile db.ChildProfile) childProfileResponse {
	return childProfileResponse{ID: uuidToString(profile.ID), DisplayName: profile.DisplayName, AvatarSeed: profile.AvatarSeed}
}

func ptrChildProfile(profile childProfileResponse) *childProfileResponse { return &profile }
