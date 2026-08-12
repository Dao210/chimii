package ldrawsync

import "testing"

func TestEmbeddedStarterKitSemanticTiers(t *testing.T) {
	_, manifest, err := EmbeddedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	creative, autoBuild, assetOnly := 0, 0, 0
	seen := map[string]bool{}
	for _, part := range manifest.Parts {
		semantics := DerivePartSemantics(part)
		if err := semantics.Validate(); err != nil {
			t.Fatal(err)
		}
		if seen[semantics.PartKey] {
			t.Fatalf("duplicate part key %q", semantics.PartKey)
		}
		seen[semantics.PartKey] = true
		if semantics.CertificationLevel == "asset_only" {
			assetOnly++
		} else {
			creative++
		}
		if semantics.AutoBuildEligible {
			autoBuild++
		}
	}
	if creative != 65 || autoBuild != 54 || assetOnly != 935 {
		t.Fatalf("creative=%d autoBuild=%d assetOnly=%d, want 65/54/935", creative, autoBuild, assetOnly)
	}
}

func TestDeriveSimplePartConnections(t *testing.T) {
	semantics := DerivePartSemantics(StarterKitPart{Rank: 1, LDrawID: "3710.dat", Name: "Plate 1 x 4", Category: "Plates"})
	if semantics.PartKey != "ldraw-3710" || semantics.StudsX != 1 || semantics.StudsZ != 4 || semantics.PlatesY != 1 {
		t.Fatalf("unexpected semantics: %#v", semantics)
	}
	if got := len(semantics.ConnectionsJSON()); got == 0 {
		t.Fatal("connections JSON is empty")
	}
}
