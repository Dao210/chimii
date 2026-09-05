package ldrawsync

import (
	"encoding/json"
	"testing"
)

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

func TestLegacyWheelPairHasReviewedMechanicalSemantics(t *testing.T) {
	holder := DerivePartSemantics(StarterKitPart{Rank: 1, LDrawID: "4600.dat", Name: "holder"})
	wheel := DerivePartSemantics(StarterKitPart{Rank: 2, LDrawID: "4624c04.dat", Name: "wheel"})
	if PartSemanticVersion < 2 || wheel.OriginYOffsetLDU != 13 {
		t.Fatalf("stale wheel semantics: version=%d wheel=%#v", PartSemanticVersion, wheel)
	}
	var holderConnectors, wheelConnectors []semanticConnector
	if err := json.Unmarshal(holder.ConnectionsJSON(), &holderConnectors); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(wheel.ConnectionsJSON(), &wheelConnectors); err != nil {
		t.Fatal(err)
	}
	pins := 0
	for _, connector := range holderConnectors {
		if connector.Kind == "wheel_pin" {
			pins++
		}
	}
	if pins != 2 || len(wheelConnectors) != 1 || wheelConnectors[0].Kind != "wheel_hole" {
		t.Fatalf("invalid wheel pair semantics: holder=%#v wheel=%#v", holderConnectors, wheelConnectors)
	}
}
