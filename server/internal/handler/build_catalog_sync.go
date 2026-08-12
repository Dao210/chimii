package handler

import (
	"errors"
	"net/http"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/ldrawsync"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type ldrawCatalogSyncResponse struct {
	Enabled           bool   `json:"enabled"`
	CanManage         bool   `json:"can_manage"`
	KitID             string `json:"kit_id"`
	CatalogVersion    string `json:"catalog_version"`
	TargetPartCount   int32  `json:"target_part_count"`
	StoredPartCount   int64  `json:"stored_part_count"`
	ProgressPartCount int32  `json:"progress_part_count"`
	Status            string `json:"status"`
	Error             string `json:"error,omitempty"`
	CreatedAt         string `json:"created_at,omitempty"`
	StartedAt         string `json:"started_at,omitempty"`
	CompletedAt       string `json:"completed_at,omitempty"`
	UpdatedAt         string `json:"updated_at,omitempty"`
}

func (h *Handler) GetLDrawCatalogSyncStatus(w http.ResponseWriter, r *http.Request) {
	member, ok := h.catalogSyncMember(w, r)
	if !ok {
		return
	}
	response, err := h.loadLDrawCatalogSyncResponse(r, roleAllowed(member.Role, "owner", "admin"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load LDraw catalog sync status")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) CreateLDrawCatalogSync(w http.ResponseWriter, r *http.Request) {
	member, ok := h.catalogSyncMember(w, r)
	if !ok {
		return
	}
	if !roleAllowed(member.Role, "owner", "admin") {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if !h.cfg.LDrawCatalogSyncEnabled || h.LDrawCatalogWorker == nil {
		writeError(w, http.StatusServiceUnavailable, "LDraw catalog sync is disabled by the operator")
		return
	}
	lock, manifest, err := ldrawsync.EmbeddedCatalog()
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "LDraw catalog manifest is unavailable")
		return
	}
	if active, err := h.Queries.GetActiveLDrawCatalogSyncJob(r.Context()); err == nil {
		response, loadErr := h.loadLDrawCatalogSyncResponse(r, true)
		if loadErr != nil {
			writeError(w, http.StatusInternalServerError, "failed to load LDraw catalog sync status")
			return
		}
		response.ProgressPartCount = active.ProgressPartCount
		writeJSON(w, http.StatusAccepted, response)
		return
	} else if !errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to start LDraw catalog sync")
		return
	}

	workspaceID := ctxWorkspaceID(r.Context())
	if workspaceID == "" {
		workspaceID = h.resolveWorkspaceID(r)
	}
	_, err = h.Queries.CreateLDrawCatalogSyncJob(r.Context(), db.CreateLDrawCatalogSyncJobParams{
		WorkspaceID:     parseUUID(workspaceID),
		RequestedBy:     parseUUID(requestUserID(r)),
		CatalogVersion:  buildstudio.ComposeLDrawCatalogVersion(lock.Release, lock.ArchiveSHA256),
		KitID:           manifest.KitID,
		TargetPartCount: int32(manifest.PartCount),
	})
	if err != nil {
		if isUniqueViolation(err) {
			if _, activeErr := h.Queries.GetActiveLDrawCatalogSyncJob(r.Context()); activeErr == nil {
				response, loadErr := h.loadLDrawCatalogSyncResponse(r, true)
				if loadErr == nil {
					writeJSON(w, http.StatusAccepted, response)
					return
				}
			}
		}
		writeError(w, http.StatusInternalServerError, "failed to start LDraw catalog sync")
		return
	}
	h.LDrawCatalogWorker.Notify()
	response, err := h.loadLDrawCatalogSyncResponse(r, true)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "LDraw catalog sync started but status could not be loaded")
		return
	}
	writeJSON(w, http.StatusAccepted, response)
}

func (h *Handler) catalogSyncMember(w http.ResponseWriter, r *http.Request) (db.Member, bool) {
	if member, ok := ctxMember(r.Context()); ok {
		return member, true
	}
	workspaceID := h.resolveWorkspaceID(r)
	return h.requireWorkspaceMember(w, r, workspaceID, "workspace not found")
}

func (h *Handler) loadLDrawCatalogSyncResponse(r *http.Request, canManage bool) (ldrawCatalogSyncResponse, error) {
	lock, manifest, err := ldrawsync.EmbeddedCatalog()
	if err != nil {
		return ldrawCatalogSyncResponse{}, err
	}
	catalogVersion := buildstudio.ComposeLDrawCatalogVersion(lock.Release, lock.ArchiveSHA256)
	stored, err := h.Queries.CountLDrawPartRevisionsByVersion(r.Context(), catalogVersion)
	if err != nil {
		return ldrawCatalogSyncResponse{}, err
	}
	response := ldrawCatalogSyncResponse{
		Enabled: h.cfg.LDrawCatalogSyncEnabled && h.LDrawCatalogWorker != nil, CanManage: canManage,
		KitID: manifest.KitID, CatalogVersion: catalogVersion, TargetPartCount: int32(manifest.PartCount),
		StoredPartCount: stored, ProgressPartCount: int32(min(stored, int64(manifest.PartCount))), Status: "idle",
	}
	release, releaseErr := h.Queries.GetLDrawCatalogReleaseByVersion(r.Context(), catalogVersion)
	if releaseErr != nil && !errors.Is(releaseErr, pgx.ErrNoRows) {
		return ldrawCatalogSyncResponse{}, releaseErr
	}
	if releaseErr == nil && release.Status == "active" && release.PartCount >= int32(manifest.PartCount) && stored >= int64(manifest.PartCount) {
		response.Status = "completed"
		response.ProgressPartCount = int32(manifest.PartCount)
	}
	job, jobErr := h.Queries.GetLatestLDrawCatalogSyncJob(r.Context())
	if errors.Is(jobErr, pgx.ErrNoRows) {
		return response, nil
	}
	if jobErr != nil {
		return ldrawCatalogSyncResponse{}, jobErr
	}
	if job.CatalogVersion != catalogVersion || job.KitID != manifest.KitID {
		return response, nil
	}
	response.Status = job.Status
	response.ProgressPartCount = job.ProgressPartCount
	if job.Error.Valid && canManage {
		response.Error = job.Error.String
	}
	response.CreatedAt = formatCatalogSyncTime(job.CreatedAt)
	response.StartedAt = formatCatalogSyncTime(job.StartedAt)
	response.CompletedAt = formatCatalogSyncTime(job.CompletedAt)
	response.UpdatedAt = formatCatalogSyncTime(job.UpdatedAt)
	return response, nil
}

func formatCatalogSyncTime(value pgtype.Timestamptz) string {
	if !value.Valid || value.Time.IsZero() {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}
