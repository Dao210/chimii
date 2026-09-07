package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/chimii-ai/chimii/server/internal/circuit"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
)

type circuitInventoryResponse struct {
	KitID          string         `json:"kit_id"`
	CatalogVersion string         `json:"catalog_version"`
	Quantities     map[string]int `json:"quantities"`
	Revision       int32          `json:"revision"`
	Confirmed      bool           `json:"confirmed"`
	CanEdit        bool           `json:"can_edit"`
	UpdatedAt      string         `json:"updated_at"`
}

func (h *Handler) ListCircuitKits(w http.ResponseWriter, r *http.Request) {
	type kit struct {
		KitID            string                   `json:"kit_id"`
		Version          string                   `json:"version"`
		Name             string                   `json:"name"`
		ConnectionSystem string                   `json:"connection_system"`
		Hardware         *circuit.HardwareProfile `json:"hardware"`
	}
	kits := []kit{}
	for _, c := range circuit.Catalogs() {
		system := c.ConnectionSystem
		if system == "" {
			system = "snap"
		}
		kits = append(kits, kit{c.KitID, c.Version, c.Name, system, c.Hardware})
	}
	writeJSON(w, 200, map[string]any{"kits": kits, "assembly_references": circuit.AssemblyReferences()})
}

func validCircuitInventory(c circuit.Catalog, quantities map[string]int, complete bool) bool {
	if quantities == nil || len(quantities) > len(c.Parts) || (complete && len(quantities) != len(c.Parts)) {
		return false
	}
	for id, n := range quantities {
		p, ok := c.Part(id)
		if !ok || n < 0 || n > p.Quantity {
			return false
		}
	}
	return true
}

func writeCircuitInventoryConflict(w http.ResponseWriter) {
	writeJSON(w, 409, map[string]string{"error": "The parts box changed; reload and check it before continuing", "code": "circuit_inventory_conflict"})
}

func savedCircuitInventory(v db.CircuitInventory, editable bool) (circuitInventoryResponse, error) {
	r := circuitInventoryResponse{KitID: v.KitID, CatalogVersion: v.CatalogVersion, Revision: v.Revision, Confirmed: true, CanEdit: editable, UpdatedAt: v.UpdatedAt.Time.UTC().Format(time.RFC3339)}
	err := json.Unmarshal(v.Quantities, &r.Quantities)
	return r, err
}

func (h *Handler) GetCircuitInventory(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	c, ok := circuit.FindCatalog(chi.URLParam(r, "kitID"))
	if !ok {
		writeError(w, 404, "kit not supported")
		return
	}
	v, err := h.Queries.GetCircuitInventory(r.Context(), db.GetCircuitInventoryParams{WorkspaceID: ws, ParentUserID: user, KitID: c.KitID})
	if errors.Is(err, pgx.ErrNoRows) {
		quantities := map[string]int{}
		for _, p := range c.Parts {
			quantities[p.ID] = 0
		}
		writeJSON(w, 200, circuitInventoryResponse{KitID: c.KitID, CatalogVersion: c.Version, Quantities: quantities, CanEdit: !child.Valid})
		return
	}
	if err != nil {
		writeError(w, 500, "could not load parts box")
		return
	}
	response, err := savedCircuitInventory(v, !child.Valid)
	if err != nil {
		writeError(w, 500, "invalid parts box")
		return
	}
	writeJSON(w, 200, response)
}

func (h *Handler) SaveCircuitInventory(w http.ResponseWriter, r *http.Request) {
	ws, user, child, ok := buildActorScope(w, r)
	if !ok {
		return
	}
	if child.Valid {
		writeJSON(w, 403, map[string]string{"error": "parent unlock required", "code": "child_capability_denied"})
		return
	}
	c, ok := circuit.FindCatalog(chi.URLParam(r, "kitID"))
	if !ok {
		writeError(w, 404, "kit not supported")
		return
	}
	var req struct {
		CatalogVersion   string         `json:"catalog_version"`
		Quantities       map[string]int `json:"quantities"`
		ExpectedRevision int32          `json:"expected_revision"`
	}
	if !decodeCircuitRequest(w, r, &req) {
		return
	}
	if req.CatalogVersion != c.Version {
		writeJSON(w, 409, map[string]string{"error": "refresh catalogue", "code": "circuit_catalog_changed"})
		return
	}
	if req.ExpectedRevision < 0 || !validCircuitInventory(c, req.Quantities, true) {
		writeError(w, 400, "invalid parts box")
		return
	}
	data, err := json.Marshal(req.Quantities)
	if err != nil {
		writeError(w, 400, "invalid parts box")
		return
	}
	tx, err := h.TxStarter.Begin(r.Context())
	if err != nil {
		writeError(w, 500, "could not save parts box")
		return
	}
	defer tx.Rollback(r.Context())
	q := h.Queries.WithTx(tx)
	if err := lockWorkspaceMemberForScopedWrite(r.Context(), q, ws, user); err != nil {
		writeError(w, 404, "workspace membership not found")
		return
	}
	var v db.CircuitInventory
	if req.ExpectedRevision == 0 {
		v, err = q.SaveCircuitInventory(r.Context(), db.SaveCircuitInventoryParams{WorkspaceID: ws, ParentUserID: user, KitID: c.KitID, CatalogVersion: c.Version, Quantities: data, ExpectedRevision: 0})
	} else {
		v, err = q.UpdateCircuitInventory(r.Context(), db.UpdateCircuitInventoryParams{WorkspaceID: ws, ParentUserID: user, KitID: c.KitID, CatalogVersion: c.Version, Quantities: data, ExpectedRevision: req.ExpectedRevision})
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeCircuitInventoryConflict(w)
		return
	}
	if err != nil {
		writeError(w, 500, "could not save parts box")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		writeError(w, 500, "could not save parts box")
		return
	}
	response, err := savedCircuitInventory(v, true)
	if err != nil {
		writeError(w, 500, "invalid parts box")
		return
	}
	writeJSON(w, 200, response)
}
