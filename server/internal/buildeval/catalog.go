package buildeval

import (
	"encoding/json"
	"fmt"

	"github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/ldrawsync"
)

type CatalogAsset struct {
	PartID        string     `json:"part_id"`
	SourceHash    string     `json:"source_hash"`
	ContentHash   string     `json:"content_hash"`
	TriangleCount int        `json:"triangle_count"`
	Bounds        [6]float64 `json:"bounds"`
}

// LongBrickCatalog is a controlled offline extension of the ten-part baseline.
// These four IDs already exist in the production manifest. Reuse its semantics
// and actual compiled assets; do not invent aliases or add owned inventory.
func LongBrickCatalog(archive string) (build.PartCatalog, []CatalogAsset, error) {
	lock, manifest, err := ldrawsync.EmbeddedCatalog()
	if err != nil {
		return nil, nil, err
	}
	if err := ldrawsync.VerifyArchive(archive, lock.ArchiveSHA256); err != nil {
		return nil, nil, err
	}
	library, err := ldrawsync.OpenLibrary(archive)
	if err != nil {
		return nil, nil, err
	}
	defer library.Close()
	ids := []string{"3010.dat", "3009.dat", "3008.dat", "2456.dat"}
	compiled, err := library.CompileParts(ids, 0)
	if err != nil {
		return nil, nil, err
	}
	catalog := build.CatalogCopy(build.StarterCatalog)
	assets := make([]CatalogAsset, 0, len(compiled))
	for _, asset := range compiled {
		var semantics ldrawsync.PartSemantics
		for _, part := range manifest.Parts {
			if part.LDrawID == asset.PartID {
				semantics = ldrawsync.DerivePartSemantics(part)
				break
			}
		}
		if err := semantics.Validate(); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", asset.PartID, err)
		}
		part := build.PartSpec{ID: semantics.PartKey, Name: semantics.Name, Category: semantics.Category,
			PopularityRank: semantics.PopularityRank, CertificationLevel: semantics.CertificationLevel,
			AutoBuildEligible: semantics.AutoBuildEligible, GeometryProfile: semantics.GeometryProfile,
			LDrawID: asset.PartID, LDrawStatus: "official", License: "CC-BY-4.0",
			StudsX: semantics.StudsX, StudsZ: semantics.StudsZ, PlatesY: semantics.PlatesY,
			Quantity: semantics.DefaultQuantity, HasTopStuds: semantics.HasTopStuds, HasBottomReceptors: semantics.HasBottomReceptors}
		for _, field := range []struct {
			raw    []byte
			target any
		}{
			{semantics.ConnectionsJSON(), &part.Connectors},
			{semantics.OccupancyJSON(), &part.Occupancy},
			{ldrawsync.BoundsJSON(asset.Bounds), &part.Bounds},
		} {
			if err := json.Unmarshal(field.raw, field.target); err != nil {
				return nil, nil, err
			}
		}
		if !build.IsShapePartEligible(part) {
			return nil, nil, fmt.Errorf("%s has no usable solid semantics", asset.PartID)
		}
		catalog[part.ID] = part
		assets = append(assets, CatalogAsset{asset.PartID, asset.LDrawSHA256, asset.ContentSHA256, asset.TriangleCount, asset.Bounds})
	}
	return catalog, assets, nil
}
