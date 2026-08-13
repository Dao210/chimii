package handler

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
)

const (
	starterKitBrowseID      = "chimii-starter-100"
	defaultCatalogPageLimit = 24
	maxCatalogPageLimit     = 50
)

type kitCatalogCursor struct {
	CatalogVersion string `json:"catalog_version"`
	Rank           int32  `json:"rank"`
	PartKey        string `json:"part_key"`
	SearchQuery    string `json:"search_query,omitempty"`
	Category       string `json:"category,omitempty"`
	Capability     string `json:"capability,omitempty"`
}

type kitCatalogPartResponse struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Category               string `json:"category"`
	PopularityRank         int32  `json:"popularity_rank"`
	CertificationLevel     string `json:"certification_level"`
	AutoBuildEligible      bool   `json:"auto_build_eligible"`
	InventoryEligible      bool   `json:"inventory_eligible"`
	GeometryProfile        string `json:"geometry_profile"`
	LDrawID                string `json:"ldraw_id"`
	LDrawStatus            string `json:"ldraw_status"`
	License                string `json:"license"`
	StudsX                 int32  `json:"studs_x"`
	StudsZ                 int32  `json:"studs_z"`
	PlatesY                int32  `json:"plates_y"`
	Quantity               int32  `json:"quantity"`
	OriginYOffsetLDU       int32  `json:"origin_y_offset_ldu,omitempty"`
	OriginCenterZOffsetLDU int32  `json:"origin_center_z_offset_ldu,omitempty"`
}

type kitCatalogPageResponse struct {
	KitID          string                   `json:"kit_id"`
	KitVersion     int32                    `json:"kit_version"`
	KitName        string                   `json:"kit_name"`
	CatalogVersion string                   `json:"catalog_version"`
	ProfileTotal   int32                    `json:"profile_total"`
	FilteredTotal  int64                    `json:"filtered_total"`
	Categories     []string                 `json:"categories"`
	Parts          []kitCatalogPartResponse `json:"parts"`
	NextCursor     string                   `json:"next_cursor,omitempty"`
}

func (h *Handler) ListBuildCatalogParts(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.Queries == nil {
		writeError(w, http.StatusServiceUnavailable, "build catalog is unavailable")
		return
	}
	release, err := h.Queries.GetLatestActiveLDrawCatalogRelease(r.Context())
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusServiceUnavailable, "build catalog has not been synchronized")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load build catalog")
		return
	}
	profile, err := h.Queries.GetActiveKitProfileForCatalog(r.Context(), db.GetActiveKitProfileForCatalogParams{
		KitID: starterKitBrowseID, CatalogVersion: release.CatalogVersion,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		writeError(w, http.StatusServiceUnavailable, "Starter Kit profile has not been synchronized")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load Starter Kit profile")
		return
	}

	search := strings.TrimSpace(r.URL.Query().Get("query"))
	if utf8.RuneCountInString(search) > 80 {
		writeError(w, http.StatusBadRequest, "catalog search must be 80 characters or fewer")
		return
	}
	category := strings.TrimSpace(r.URL.Query().Get("category"))
	if utf8.RuneCountInString(category) > 80 {
		writeError(w, http.StatusBadRequest, "catalog category is invalid")
		return
	}
	capability := strings.TrimSpace(r.URL.Query().Get("capability"))
	if capability == "all" {
		capability = ""
	}
	if capability != "" && capability != "auto_build" && capability != "inventory" && capability != "preview" {
		writeError(w, http.StatusBadRequest, "catalog capability is invalid")
		return
	}
	limit := defaultCatalogPageLimit
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		parsed, parseErr := strconv.Atoi(rawLimit)
		if parseErr != nil || parsed < 1 || parsed > maxCatalogPageLimit {
			writeError(w, http.StatusBadRequest, "catalog limit must be between 1 and 50")
			return
		}
		limit = parsed
	}
	cursor, err := decodeKitCatalogCursor(r.URL.Query().Get("cursor"), release.CatalogVersion, search, category, capability)
	if err != nil {
		writeError(w, http.StatusBadRequest, "catalog cursor is invalid")
		return
	}

	filter := db.CountKitCatalogPartsParams{
		KitID: profile.KitID, KitVersion: profile.Version, CatalogVersion: profile.CatalogVersion,
		SearchQuery: search, CategoryFilter: category, CapabilityFilter: capability,
	}
	total, err := h.Queries.CountKitCatalogParts(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to count Starter Kit parts")
		return
	}
	categories, err := h.Queries.ListKitCatalogCategories(r.Context(), db.ListKitCatalogCategoriesParams{
		KitID: profile.KitID, KitVersion: profile.Version, CatalogVersion: profile.CatalogVersion,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load Starter Kit categories")
		return
	}
	rows, err := h.Queries.ListKitCatalogParts(r.Context(), db.ListKitCatalogPartsParams{
		KitID: profile.KitID, KitVersion: profile.Version, CatalogVersion: profile.CatalogVersion,
		SearchQuery: search, CategoryFilter: category, CapabilityFilter: capability,
		AfterRank: cursor.Rank, AfterPartKey: cursor.PartKey, PageLimit: int32(limit + 1),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load Starter Kit parts")
		return
	}

	nextCursor := ""
	if len(rows) > limit {
		last := rows[limit-1]
		nextCursor = encodeKitCatalogCursor(kitCatalogCursor{
			CatalogVersion: profile.CatalogVersion, Rank: last.PopularityRank, PartKey: last.PartKey,
			SearchQuery: search, Category: category, Capability: capability,
		})
		rows = rows[:limit]
	}
	parts := make([]kitCatalogPartResponse, 0, len(rows))
	for _, row := range rows {
		parts = append(parts, kitCatalogPartResponse{
			ID: row.PartKey, Name: row.Name, Category: row.Category, PopularityRank: row.PopularityRank,
			CertificationLevel: row.CertificationLevel, AutoBuildEligible: row.AutoBuildEligible,
			InventoryEligible: row.CertificationLevel != "asset_only", GeometryProfile: row.GeometryProfile,
			LDrawID: row.LdrawPartID, LDrawStatus: "official", License: "CC-BY-4.0",
			StudsX: row.StudsX, StudsZ: row.StudsZ, PlatesY: row.PlatesY, Quantity: row.DefaultQuantity,
			OriginYOffsetLDU: row.OriginYOffsetLdu, OriginCenterZOffsetLDU: row.OriginCenterZOffsetLdu,
		})
	}
	writeJSON(w, http.StatusOK, kitCatalogPageResponse{
		KitID: profile.KitID, KitVersion: profile.Version, KitName: profile.Name,
		CatalogVersion: profile.CatalogVersion, ProfileTotal: profile.PartCount, FilteredTotal: total,
		Categories: categories, Parts: parts, NextCursor: nextCursor,
	})
}

func decodeKitCatalogCursor(raw, catalogVersion, search, category, capability string) (kitCatalogCursor, error) {
	if strings.TrimSpace(raw) == "" {
		return kitCatalogCursor{}, nil
	}
	if len(raw) > 1024 {
		return kitCatalogCursor{}, errors.New("cursor is too large")
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return kitCatalogCursor{}, err
	}
	var cursor kitCatalogCursor
	decoder := json.NewDecoder(strings.NewReader(string(decoded)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cursor); err != nil {
		return kitCatalogCursor{}, err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return kitCatalogCursor{}, errors.New("cursor has trailing content")
	}
	if cursor.CatalogVersion != catalogVersion || cursor.Rank <= 0 || strings.TrimSpace(cursor.PartKey) == "" ||
		cursor.SearchQuery != search || cursor.Category != category || cursor.Capability != capability {
		return kitCatalogCursor{}, errors.New("cursor does not match active catalog")
	}
	return cursor, nil
}

func encodeKitCatalogCursor(cursor kitCatalogCursor) string {
	raw, _ := json.Marshal(cursor)
	return base64.RawURLEncoding.EncodeToString(raw)
}
