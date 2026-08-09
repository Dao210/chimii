package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const (
	maxBrickInventoryItems    = 60
	maxBrickInventoryQuantity = 999
)

type buildCatalogColorResponse struct {
	Code int    `json:"code"`
	Name string `json:"name"`
	Hex  string `json:"hex"`
}

type buildCatalogResponse struct {
	CatalogVersion string                      `json:"catalog_version"`
	Parts          []buildstudio.PartSpec      `json:"parts"`
	Colors         []buildCatalogColorResponse `json:"colors"`
}

type brickInventoryItemRequest struct {
	PartID   string `json:"part_id"`
	Color    int    `json:"color"`
	Quantity int    `json:"quantity"`
}

type saveBrickInventoryRequest struct {
	ExpectedRevision int32                       `json:"expected_revision"`
	Items            []brickInventoryItemRequest `json:"items"`
}

type brickInventoryResponse struct {
	Configured     bool                        `json:"configured"`
	CatalogVersion string                      `json:"catalog_version"`
	Revision       int32                       `json:"revision"`
	Items          []buildstudio.InventoryItem `json:"items"`
	UpdatedAt      string                      `json:"updated_at,omitempty"`
}

func (h *Handler) GetBuildCatalog(w http.ResponseWriter, _ *http.Request) {
	parts := make([]buildstudio.PartSpec, 0, len(buildstudio.StarterCatalog))
	for _, part := range buildstudio.StarterCatalog {
		parts = append(parts, part)
	}
	sort.Slice(parts, func(i, j int) bool {
		if parts[i].Category != parts[j].Category {
			return parts[i].Category < parts[j].Category
		}
		return parts[i].ID < parts[j].ID
	})
	writeJSON(w, http.StatusOK, buildCatalogResponse{
		CatalogVersion: buildstudio.CatalogVersion,
		Parts:          parts,
		Colors: []buildCatalogColorResponse{
			{Code: 1, Name: "Blue", Hex: "#1e5aa8"},
			{Code: 2, Name: "Green", Hex: "#00852b"},
			{Code: 4, Name: "Red", Hex: "#b40000"},
			{Code: 14, Name: "Yellow", Hex: "#fac80a"},
			{Code: 15, Name: "White", Hex: "#f4f4f4"},
			{Code: 71, Name: "Light gray", Hex: "#969696"},
		},
	})
}

func (h *Handler) GetBrickInventory(w http.ResponseWriter, r *http.Request) {
	workspaceID, _, ok := buildActorIDs(w, r)
	if !ok {
		return
	}
	response, err := loadBrickInventoryResponse(r.Context(), h.Queries, workspaceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to load brick inventory")
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) PutBrickInventory(w http.ResponseWriter, r *http.Request) {
	workspaceID, userID, _, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	var req saveBrickInventoryRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 32<<10)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ExpectedRevision < 0 || len(req.Items) > maxBrickInventoryItems {
		writeError(w, http.StatusBadRequest, "invalid brick inventory")
		return
	}
	items, err := normalizeBrickInventoryItems(req.Items)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), qtx, workspaceID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace membership not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
		return
	}
	if err := qtx.LockBrickInventoryForWorkspace(r.Context(), uuidToString(workspaceID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
		return
	}
	currentRevision := int32(0)
	current, lookupErr := qtx.GetBrickInventoryByWorkspace(r.Context(), workspaceID)
	if lookupErr == nil {
		currentRevision = current.Revision
	} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
		return
	}
	if req.ExpectedRevision != currentRevision {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "brick inventory changed in another session", "code": "BRICK_INVENTORY_REVISION_CONFLICT", "revision": currentRevision,
		})
		return
	}
	inventory, err := qtx.SaveBrickInventory(r.Context(), db.SaveBrickInventoryParams{
		WorkspaceID: workspaceID, CatalogVersion: buildstudio.CatalogVersion, UpdatedBy: userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
		return
	}
	if err := qtx.DeleteBrickInventoryItems(r.Context(), inventory.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
		return
	}
	for _, item := range items {
		if err := qtx.CreateBrickInventoryItem(r.Context(), db.CreateBrickInventoryItemParams{
			InventoryID: inventory.ID, PartKey: item.PartID, ColorCode: int32(item.Color), Quantity: int32(item.Quantity),
		}); err != nil {
			writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
			return
		}
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save brick inventory")
		return
	}
	response := brickInventoryResponse{
		Configured: true, CatalogVersion: inventory.CatalogVersion, Revision: inventory.Revision, Items: items,
	}
	if inventory.UpdatedAt.Valid {
		response.UpdatedAt = inventory.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) DeleteBrickInventory(w http.ResponseWriter, r *http.Request) {
	workspaceID, userID, _, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	expectedRevisionValue, err := strconv.ParseInt(r.URL.Query().Get("expected_revision"), 10, 32)
	if err != nil || expectedRevisionValue < 0 {
		writeError(w, http.StatusBadRequest, "expected_revision is required")
		return
	}
	expectedRevision := int32(expectedRevisionValue)
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset brick inventory")
		return
	}
	defer tx.Rollback(r.Context())
	qtx := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), qtx, workspaceID, userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeError(w, http.StatusNotFound, "workspace membership not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to reset brick inventory")
		return
	}
	if err := qtx.LockBrickInventoryForWorkspace(r.Context(), uuidToString(workspaceID)); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset brick inventory")
		return
	}
	currentRevision := int32(0)
	current, lookupErr := qtx.GetBrickInventoryByWorkspace(r.Context(), workspaceID)
	if lookupErr == nil {
		currentRevision = current.Revision
	} else if !errors.Is(lookupErr, pgx.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "failed to reset brick inventory")
		return
	}
	if expectedRevision != currentRevision {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error": "brick inventory changed in another session", "code": "BRICK_INVENTORY_REVISION_CONFLICT", "revision": currentRevision,
		})
		return
	}
	inventory, err := qtx.ResetBrickInventory(r.Context(), db.ResetBrickInventoryParams{
		WorkspaceID: workspaceID, CatalogVersion: buildstudio.CatalogVersion, UpdatedBy: userID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset brick inventory")
		return
	}
	if err := qtx.DeleteBrickInventoryItems(r.Context(), inventory.ID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset brick inventory")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to reset brick inventory")
		return
	}
	response := brickInventoryResponse{
		Configured: false, CatalogVersion: inventory.CatalogVersion, Revision: inventory.Revision, Items: []buildstudio.InventoryItem{},
	}
	if inventory.UpdatedAt.Valid {
		response.UpdatedAt = inventory.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	writeJSON(w, http.StatusOK, response)
}

func normalizeBrickInventoryItems(requestItems []brickInventoryItemRequest) ([]buildstudio.InventoryItem, error) {
	items := make([]buildstudio.InventoryItem, 0, len(requestItems))
	seen := make(map[string]bool, len(requestItems))
	for _, item := range requestItems {
		if _, ok := buildstudio.StarterCatalog[item.PartID]; !ok {
			return nil, errors.New("brick inventory contains an unknown part")
		}
		if !buildstudio.IsAllowedColor(item.Color) {
			return nil, errors.New("brick inventory contains an unknown color")
		}
		if item.Quantity < 0 || item.Quantity > maxBrickInventoryQuantity {
			return nil, errors.New("brick quantity must be between 0 and 999")
		}
		if item.Quantity == 0 {
			continue
		}
		key := item.PartID + ":" + strconv.Itoa(item.Color)
		if seen[key] {
			return nil, errors.New("brick inventory contains a duplicate part and color")
		}
		seen[key] = true
		items = append(items, buildstudio.InventoryItem{PartID: item.PartID, Color: item.Color, Quantity: item.Quantity})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].PartID != items[j].PartID {
			return items[i].PartID < items[j].PartID
		}
		return items[i].Color < items[j].Color
	})
	return items, nil
}

func loadBrickInventoryResponse(ctx context.Context, queries *db.Queries, workspaceID pgtype.UUID) (brickInventoryResponse, error) {
	inventory, err := queries.GetBrickInventoryByWorkspace(ctx, workspaceID)
	if errors.Is(err, pgx.ErrNoRows) {
		return unlimitedBrickInventoryResponse(), nil
	}
	if err != nil {
		return brickInventoryResponse{}, err
	}
	rows, err := queries.ListBrickInventoryItems(ctx, inventory.ID)
	if err != nil {
		return brickInventoryResponse{}, err
	}
	items := make([]buildstudio.InventoryItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildstudio.InventoryItem{PartID: row.PartKey, Color: int(row.ColorCode), Quantity: int(row.Quantity)})
	}
	response := brickInventoryResponse{Configured: inventory.Configured, CatalogVersion: inventory.CatalogVersion, Revision: inventory.Revision, Items: items}
	if inventory.UpdatedAt.Valid {
		response.UpdatedAt = inventory.UpdatedAt.Time.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return response, nil
}

func loadBrickInventorySnapshot(ctx context.Context, queries *db.Queries, workspaceID pgtype.UUID) (buildstudio.InventorySnapshot, error) {
	response, err := loadBrickInventoryResponse(ctx, queries, workspaceID)
	if err != nil {
		return buildstudio.InventorySnapshot{}, err
	}
	return buildstudio.NewInventorySnapshot(response.Configured, response.Revision, response.Items), nil
}

func unlimitedBrickInventoryResponse() brickInventoryResponse {
	return brickInventoryResponse{Configured: false, CatalogVersion: buildstudio.CatalogVersion, Revision: 0, Items: []buildstudio.InventoryItem{}}
}
