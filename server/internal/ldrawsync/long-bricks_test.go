package ldrawsync

import (
	"encoding/binary"
	"encoding/json"
	"os"
	"testing"
)

func TestReviewedLongBrickSemantics(t *testing.T) {
	_, manifest, err := EmbeddedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string][2]int{"3010.dat": {4, 1}, "3009.dat": {6, 1}, "3008.dat": {8, 1}, "2456.dat": {6, 2}}
	for _, p := range manifest.Parts {
		size, ok := want[p.LDrawID]
		if !ok {
			continue
		}
		s := DerivePartSemantics(p)
		if s.StudsX != size[0] || s.StudsZ != size[1] || s.PlatesY != 3 || !s.AutoBuildEligible || s.PartKey != "ldraw-"+p.LDrawID[:len(p.LDrawID)-4] {
			t.Errorf("%s: semantics disagree with reviewed LDraw axes: %+v", p.LDrawID, s)
		}
		var connectors []semanticConnector
		if err := json.Unmarshal(s.ConnectionsJSON(), &connectors); err != nil {
			t.Fatal(err)
		}
		if len(connectors) != size[0]*size[1]*2 {
			t.Fatal("incomplete connector grid")
		}
		delete(want, p.LDrawID)
	}
	if len(want) != 0 {
		t.Fatal("missing long bricks", want)
	}
}

func TestLongBrickAssetsMatchReviewedSemantics(t *testing.T) {
	archive := os.Getenv("CHIMII_LDRAW_ARCHIVE")
	if archive == "" {
		t.Skip("set CHIMII_LDRAW_ARCHIVE to the pinned official archive")
	}
	lock, manifest, err := EmbeddedCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyArchive(archive, lock.ArchiveSHA256); err != nil {
		t.Fatal(err)
	}
	library, err := OpenLibrary(archive)
	if err != nil {
		t.Fatal(err)
	}
	defer library.Close()
	ids := []string{"3010.dat", "3009.dat", "3008.dat", "2456.dat"}
	parts, err := library.CompileParts(ids, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, asset := range parts {
		var s PartSemantics
		for _, p := range manifest.Parts {
			if p.LDrawID == asset.PartID {
				s = DerivePartSemantics(p)
				break
			}
		}
		b := asset.Bounds
		if b[3]-b[0] != float64(s.StudsX*20) || b[5]-b[2] != float64(s.StudsZ*20) || b[4] != float64(s.PlatesY*8) || b[1] != -4 {
			t.Errorf("%s: asset bounds %v disagree with %dx%dx%d semantics", asset.PartID, b, s.StudsX, s.StudsZ, s.PlatesY)
		}
		if len(asset.Content) < 20 || string(asset.Content[:4]) != "glTF" || binary.LittleEndian.Uint32(asset.Content[8:12]) != uint32(len(asset.Content)) || asset.TriangleCount <= 0 {
			t.Fatal("invalid or empty GLB", asset.PartID)
		}
	}
}
