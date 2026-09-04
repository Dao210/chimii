package ldrawsync

import "testing"

func TestEmbeddedCatalog(t *testing.T) {
	lock, manifest, err := EmbeddedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if lock.ArchiveSHA256 != "d2a695868ed2b3957c45b022a6451908edab22cc043179dd61d18dd382b35e11" {
		t.Fatalf("unexpected archive hash: %s", lock.ArchiveSHA256)
	}
	if manifest.KitID != "chimii-starter-1000-v1" || manifest.PartCount != 1000 {
		t.Fatalf("unexpected Starter Kit: %s (%d parts)", manifest.KitID, manifest.PartCount)
	}

	seen := map[string]bool{}
	for _, part := range manifest.Parts {
		seen[part.LDrawID] = true
	}
	for _, required := range lock.RootParts {
		if !seen[required] {
			t.Errorf("embedded fallback part %s is missing from Starter Kit", required)
		}
	}
}

func TestParseStarterKitRejectsCountMismatch(t *testing.T) {
	_, err := ParseStarterKit([]byte(`{
		"schema_version": 1,
		"kit_id": "test",
		"part_count": 2,
		"selection": {},
		"parts": [{"rank": 1, "ldraw_id": "3001.dat", "name": "Brick"}]
	}`))
	if err == nil {
		t.Fatal("expected part count mismatch")
	}
}
