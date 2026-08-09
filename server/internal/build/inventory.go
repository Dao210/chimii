package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

type inventoryKey struct {
	partID string
	color  int
}

// UnlimitedInventory is the product default when a workspace has never saved
// an inventory. It intentionally removes the Starter Kit quantity ceiling.
func UnlimitedInventory() InventorySnapshot {
	return NewInventorySnapshot(false, 0, nil)
}

// NewInventorySnapshot normalizes order and computes a stable audit hash.
func NewInventorySnapshot(configured bool, revision int32, items []InventoryItem) InventorySnapshot {
	// Keep the JSON contract stable: an empty inventory is [] rather than null.
	// Build plans are persisted as JSON and consumed by older installed clients,
	// so the wire shape must not depend on whether the caller passed a nil slice.
	normalized := append([]InventoryItem{}, items...)
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].PartID != normalized[j].PartID {
			return normalized[i].PartID < normalized[j].PartID
		}
		return normalized[i].Color < normalized[j].Color
	})
	snapshot := InventorySnapshot{
		Configured: configured, CatalogVersion: CatalogVersion, Revision: revision, Items: normalized,
	}
	payload, _ := json.Marshal(struct {
		Configured     bool            `json:"configured"`
		CatalogVersion string          `json:"catalog_version"`
		Revision       int32           `json:"revision"`
		Items          []InventoryItem `json:"items"`
	}{snapshot.Configured, snapshot.CatalogVersion, snapshot.Revision, snapshot.Items})
	sum := sha256.Sum256(payload)
	snapshot.ContentHash = hex.EncodeToString(sum[:])
	return snapshot
}

func (snapshot InventorySnapshot) quantities() map[inventoryKey]int {
	quantities := make(map[inventoryKey]int, len(snapshot.Items))
	for _, item := range snapshot.Items {
		if item.Quantity <= 0 {
			continue
		}
		quantities[inventoryKey{partID: item.PartID, color: item.Color}] += item.Quantity
	}
	return quantities
}

// resolveInventoryColors keeps the structural template deterministic while
// reassigning its color roles to colors the user actually owns. The desired
// color is preferred; otherwise the color with the greatest remaining count
// for that part wins. Final validation still re-checks every count.
func resolveInventoryColors(placements []Placement, snapshot InventorySnapshot) []Placement {
	resolved := append([]Placement(nil), placements...)
	if !snapshot.Configured {
		return resolved
	}
	remaining := snapshot.quantities()
	roleColors := map[int]int{}
	colors := AllowedColorCodes()
	for index := range resolved {
		placement := &resolved[index]
		candidates := make([]int, 0, len(colors)+2)
		if mapped, ok := roleColors[placement.Color]; ok {
			candidates = append(candidates, mapped)
		}
		candidates = append(candidates, placement.Color)
		sort.SliceStable(colors, func(i, j int) bool {
			left := remaining[inventoryKey{partID: placement.PartID, color: colors[i]}]
			right := remaining[inventoryKey{partID: placement.PartID, color: colors[j]}]
			if left != right {
				return left > right
			}
			return colors[i] < colors[j]
		})
		candidates = append(candidates, colors...)
		seen := map[int]bool{}
		for _, color := range candidates {
			if seen[color] {
				continue
			}
			seen[color] = true
			key := inventoryKey{partID: placement.PartID, color: color}
			if remaining[key] <= 0 {
				continue
			}
			roleColors[placement.Color] = color
			placement.Color = color
			remaining[key]--
			break
		}
	}
	return resolved
}
