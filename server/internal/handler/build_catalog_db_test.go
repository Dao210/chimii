package handler

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	"github.com/chimii-ai/chimii/server/internal/ldrawsync"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
)

// Exercises the actual sync -> DB catalog -> inventory -> compiler -> MPD path.
// The helper isolates all writes in a temporary schema in the explicit test DB.
func TestBuildLongBrickCatalogRoundTrip(t *testing.T) {
	archive := os.Getenv("CHIMII_LDRAW_ARCHIVE")
	if archive == "" {
		t.Skip("set CHIMII_LDRAW_ARCHIVE to the pinned official archive")
	}
	h, pool, _ := buildTestDB(t)
	ctx := context.Background()
	lock, manifest, err := ldrawsync.EmbeddedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	wanted := map[string]bool{"3010.dat": true, "3009.dat": true, "3008.dat": true, "2456.dat": true}
	// Activation also publishes the existing Starter Kit 100 profile. Preserve
	// its real 100 ranked members; all four reviewed long bricks are in it.
	manifest.Parts, manifest.PartCount = manifest.Parts[:100], 100
	if err := ldrawsync.SyncCatalog(ctx, pool, lock, manifest, archive, nil); err != nil {
		t.Fatal(err)
	}
	catalog, err := loadActiveBuildCatalog(ctx, h.Queries)
	if err != nil || len(catalog.Parts) < 4 {
		t.Fatalf("load synchronized catalog: %d, %v", len(catalog.Parts), err)
	}
	for id, p := range catalog.Parts {
		if !wanted[p.LDrawID] {
			continue
		}
		delete(wanted, p.LDrawID)
		asset, err := h.Queries.GetLDrawPartRevisionByVersionAndPartID(ctx, db.GetLDrawPartRevisionByVersionAndPartIDParams{CatalogVersion: catalog.Version, PartID: p.LDrawID})
		if err != nil || len(asset.Payload) < 20 || string(asset.Payload[:4]) != "glTF" {
			t.Fatalf("missing GLB for %s: %v", id, err)
		}
		for _, rotation := range []int{0, 90} {
			sx, sz := p.StudsX, p.StudsZ
			if rotation == 90 {
				sx, sz = sz, sx
			}
			recipe := buildstudio.AssemblyRecipe{Version: 3, Subject: "custom", Archetype: "custom", Title: "Long brick acceptance", Design: &buildstudio.DesignSpec{Version: 1, Mode: "static", Shapes: []buildstudio.ShapeNode{{ID: "solid", Kind: "box", Operation: "add", Size: buildstudio.DesignVector{X: sx, Y: 6, Z: sz}, Color: 1}}}, Constraints: buildstudio.RecipeConstraints{ExactColors: true, PartCount: 2}}
			items, err := normalizeBrickInventoryItems([]brickInventoryItemRequest{{PartID: id, Color: 1, Quantity: 2}}, catalog.Parts)
			if err != nil || len(items) != 1 {
				t.Fatalf("inventory selection %s: %v", id, err)
			}
			inv := buildstudio.NewInventorySnapshot(true, 1, items)
			for repeat := 0; repeat < 2; repeat++ {
				result, err := buildstudio.CompileDesign(ctx, recipe, inv, catalog.Version, catalog.Parts, time.Unix(0, 0))
				if err != nil || !result.Plan.Validation.Buildable || len(result.Plan.Placements) != 2 || !strings.Contains(result.MPD, p.LDrawID) {
					t.Fatalf("compile/MPD %s rotation %d: %v", id, rotation, err)
				}
				for _, placed := range result.Plan.Placements {
					if placed.PartID != id || placed.Rotation != rotation {
						t.Fatalf("lost part identity or orientation: %+v", placed)
					}
				}
			}
			if !reflect.DeepEqual(inv.Items, items) {
				t.Fatal("compilation consumed inventory")
			}
			// The other catalog entries are selectable, not implicitly owned.
			recipe.Design.Shapes[0].Size.Y = 9
			recipe.Constraints.PartCount = 3
			if _, err := buildstudio.CompileDesign(ctx, recipe, inv, catalog.Version, catalog.Parts, time.Unix(0, 0)); err == nil {
				t.Fatal("catalog defaults replenished configured inventory")
			}
		}
	}
	if len(wanted) != 0 {
		t.Fatal("reviewed long bricks missing from synchronized catalog", wanted)
	}
	// Mixed semantic revisions must not expose old-axis rows to new searches.
	if _, err := pool.Exec(ctx, "UPDATE part_catalog_revision SET semantic_version=2 WHERE catalog_version=$1 AND part_key='ldraw-3010'", catalog.Version); err != nil {
		t.Fatal(err)
	}
	stale, err := loadActiveBuildCatalog(ctx, h.Queries)
	if err != nil || len(stale.Parts) != len(buildstudio.StarterCatalog) {
		t.Fatalf("mixed semantics did not use the reviewed fallback: %v", err)
	}
	if _, exists := stale.Parts["ldraw-3010"]; exists {
		t.Fatal("stale long-brick axes escaped the semantic revision gate")
	}
}
