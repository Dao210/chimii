package handler

import "testing"

func TestKitCatalogCursorRoundTrip(t *testing.T) {
	want := kitCatalogCursor{CatalogVersion: "catalog-v1", Rank: 24, PartKey: "ldraw-3024", SearchQuery: "brick", Category: "Bricks", Capability: "auto_build"}
	raw := encodeKitCatalogCursor(want)
	got, err := decodeKitCatalogCursor(raw, want.CatalogVersion, want.SearchQuery, want.Category, want.Capability)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("cursor = %#v, want %#v", got, want)
	}
}

func TestKitCatalogCursorRejectsVersionDrift(t *testing.T) {
	raw := encodeKitCatalogCursor(kitCatalogCursor{CatalogVersion: "catalog-v1", Rank: 1, PartKey: "brick"})
	if _, err := decodeKitCatalogCursor(raw, "catalog-v2", "", "", ""); err == nil {
		t.Fatal("expected catalog version mismatch")
	}
}

func TestKitCatalogCursorRejectsMalformedValue(t *testing.T) {
	for _, raw := range []string{"not-base64!", encodeKitCatalogCursor(kitCatalogCursor{CatalogVersion: "catalog-v1"})} {
		if _, err := decodeKitCatalogCursor(raw, "catalog-v1", "", "", ""); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestKitCatalogCursorRejectsFilterDrift(t *testing.T) {
	raw := encodeKitCatalogCursor(kitCatalogCursor{CatalogVersion: "catalog-v1", Rank: 1, PartKey: "brick", SearchQuery: "brick"})
	if _, err := decodeKitCatalogCursor(raw, "catalog-v1", "plate", "", ""); err == nil {
		t.Fatal("expected search filter mismatch")
	}
}
