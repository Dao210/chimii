package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/ldrawsync"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/jackc/pgx/v5"
)

type resolvedBuildCatalog struct {
	Version string
	Source  buildCatalogSourceResponse
	Parts   buildstudio.PartCatalog
}

func fallbackBuildCatalog() resolvedBuildCatalog {
	return resolvedBuildCatalog{
		Version: buildstudio.CatalogVersion,
		Source: buildCatalogSourceResponse{
			Release: buildstudio.CatalogRelease, ArchiveSHA256: buildstudio.CatalogArchiveSHA256,
			SourceURL: buildstudio.CatalogSourceURL,
		},
		Parts: buildstudio.CatalogCopy(buildstudio.StarterCatalog),
	}
}

func loadActiveBuildCatalog(ctx context.Context, queries *db.Queries) (resolvedBuildCatalog, error) {
	if queries == nil {
		return fallbackBuildCatalog(), nil
	}
	release, err := queries.GetLatestActiveLDrawCatalogRelease(ctx)
	if errors.Is(err, pgx.ErrNoRows) {
		return fallbackBuildCatalog(), nil
	}
	if err != nil {
		return resolvedBuildCatalog{}, err
	}
	parts, err := loadBuildCatalogParts(ctx, queries, release.CatalogVersion)
	if err != nil {
		return resolvedBuildCatalog{}, err
	}
	if len(parts) == 0 {
		parts = buildstudio.CatalogCopy(buildstudio.StarterCatalog)
	}
	return resolvedBuildCatalog{
		Version: release.CatalogVersion,
		Source:  buildCatalogSourceResponse{Release: release.Release, ArchiveSHA256: release.ArchiveSha256, SourceURL: release.SourceUrl},
		Parts:   parts,
	}, nil
}

func loadBuildCatalogParts(ctx context.Context, queries *db.Queries, catalogVersion string) (buildstudio.PartCatalog, error) {
	if queries == nil || catalogVersion == "" {
		return buildstudio.CatalogCopy(buildstudio.StarterCatalog), nil
	}
	rows, err := queries.ListCreativeCatalogPartsByVersion(ctx, catalogVersion)
	if err != nil {
		return nil, err
	}
	// Catalog sync updates rows incrementally. Never expose a mixed or stale
	// semantic generation to the compiler: keep serving the small baked,
	// reviewed catalog until every creative row reaches the required version.
	for _, row := range rows {
		if row.SemanticVersion < ldrawsync.PartSemanticVersion {
			return buildstudio.CatalogCopy(buildstudio.StarterCatalog), nil
		}
	}
	parts := make(buildstudio.PartCatalog, len(rows))
	for _, row := range rows {
		part := buildstudio.PartSpec{
			ID: row.PartKey, Name: row.Name, Category: row.Category,
			PopularityRank: int(row.PopularityRank), CertificationLevel: row.CertificationLevel,
			AutoBuildEligible: row.AutoBuildEligible, GeometryProfile: row.GeometryProfile,
			LDrawID: row.LdrawPartID, LDrawStatus: "official", License: "CC-BY-4.0",
			StudsX: int(row.StudsX), StudsZ: int(row.StudsZ), PlatesY: int(row.PlatesY),
			Quantity: int(row.DefaultQuantity), HasTopStuds: row.HasTopStuds,
			HasBottomReceptors: row.HasBottomReceptors, OriginYOffsetLDU: int(row.OriginYOffsetLdu),
			OriginCenterZOffsetLDU: int(row.OriginCenterZOffsetLdu),
		}
		if err := json.Unmarshal(row.BoundsJson, &part.Bounds); err != nil {
			return nil, fmt.Errorf("decode bounds for %s: %w", row.PartKey, err)
		}
		if err := json.Unmarshal(row.ConnectionsJson, &part.Connectors); err != nil {
			return nil, fmt.Errorf("decode connectors for %s: %w", row.PartKey, err)
		}
		if err := json.Unmarshal(row.OccupancyJson, &part.Occupancy); err != nil {
			return nil, fmt.Errorf("decode occupancy for %s: %w", row.PartKey, err)
		}
		parts[row.PartKey] = part
	}
	if len(parts) == 0 {
		return buildstudio.CatalogCopy(buildstudio.StarterCatalog), nil
	}
	return parts, nil
}

func sortedBuildCatalogParts(parts buildstudio.PartCatalog) []buildstudio.PartSpec {
	result := make([]buildstudio.PartSpec, 0, len(parts))
	for _, part := range parts {
		result = append(result, part)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].PopularityRank != result[j].PopularityRank {
			return result[i].PopularityRank < result[j].PopularityRank
		}
		return result[i].ID < result[j].ID
	})
	return result
}
