package handler

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
)

func TestNormalizeBrickInventoryItems(t *testing.T) {
	items, err := normalizeBrickInventoryItems([]brickInventoryItemRequest{
		{PartID: "plate-1x2", Color: 14, Quantity: 0},
		{PartID: "brick-2x4", Color: 4, Quantity: 6},
		{PartID: "brick-1x1", Color: 1, Quantity: 2},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].PartID != "brick-1x1" || items[1].PartID != "brick-2x4" {
		t.Fatalf("normalized items = %#v", items)
	}
	if items[0].Quantity != 2 || items[1].Quantity != 6 {
		t.Fatalf("normalized quantities = %#v", items)
	}
}

func TestNormalizeBrickInventoryItemsRejectsInvalidEntries(t *testing.T) {
	for name, items := range map[string][]brickInventoryItemRequest{
		"unknown part":  {{PartID: "imaginary", Color: 4, Quantity: 1}},
		"unknown color": {{PartID: "brick-2x4", Color: 999, Quantity: 1}},
		"too many":      {{PartID: "brick-2x4", Color: 4, Quantity: 1000}},
		"duplicate": {
			{PartID: "brick-2x4", Color: 4, Quantity: 1},
			{PartID: "brick-2x4", Color: 4, Quantity: 2},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := normalizeBrickInventoryItems(items); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestGetBuildCatalogReturnsSourceMetadata(t *testing.T) {
	h := &Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/build/catalog", nil)
	h.GetBuildCatalog(w, r)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, expected %d", resp.StatusCode, http.StatusOK)
	}
	body := strings.TrimSpace(string(mustReadAll(t, resp.Body)))
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	source, ok := payload["catalog_source"].(map[string]any)
	if !ok {
		t.Fatalf("missing catalog_source: %#v", payload["catalog_source"])
	}
	if source["release"] != buildstudio.CatalogRelease {
		t.Fatalf("release = %v, want %q", source["release"], buildstudio.CatalogRelease)
	}
	if source["archive_sha256"] != buildstudio.CatalogArchiveSHA256 {
		t.Fatalf("archive sha mismatch: %v", source["archive_sha256"])
	}
	if source["source_url"] != buildstudio.CatalogSourceURL {
		t.Fatalf("source url mismatch: %v", source["source_url"])
	}
}

func mustReadAll(t *testing.T, body interface{ Read(p []byte) (int, error) }) []byte {
	t.Helper()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return data
}
