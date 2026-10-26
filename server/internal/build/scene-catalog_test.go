package build

import "testing"

func TestSceneCatalogDoesNotAdvertiseLongBeamsFromAssetOnlyEntries(t *testing.T) {
	fallback := DescribeSceneCatalog(StarterCatalog)
	if fallback.FullPierGridBeams || fallback.MaxRectangularLengthStuds != 4 {
		t.Fatalf("fallback advertised unsupported spans: %+v", fallback)
	}
	full := DescribeSceneCatalog(sceneCatalog())
	if !full.FullPierGridBeams || full.MaxRectangularLengthStuds != 8 {
		t.Fatalf("reviewed long bricks missing: %+v", full)
	}
	catalog := CatalogCopy(StarterCatalog)
	catalog["display-only-long"] = PartSpec{ID: "display-only-long", StudsX: 32, StudsZ: 2, PlatesY: 3, CertificationLevel: "asset_only"}
	if got := DescribeSceneCatalog(catalog); got.FullPierGridBeams || got.MaxRectangularLengthStuds != 4 {
		t.Fatal("render-only part became a structural beam")
	}
}
