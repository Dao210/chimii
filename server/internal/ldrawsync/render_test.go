package ldrawsync

import (
	"archive/zip"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestRenderLinesPreserveTransformsColorsAndMeshBounds(t *testing.T) {
	sources := map[string]string{
		"parts/test.dat": "1 4 10 20 30 -1 0 0 0 1 0 0 0 1 child.dat\n",
		"p/child.dat":    "3 16 0 0 0 2 0 0 0 2 0\n2 24 0 0 0 2 0 0\n5 24 0 0 0 2 0 0 0 10000 0 2 10000 0\n2 16 0 0 0 0 2 0\n",
	}
	library := ArchiveLibrary{files: map[string]*zip.File{}, cache: map[string]*sourceFile{}}
	for path, raw := range sources {
		lines, err := parseSource([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		library.files[path] = &zip.File{}
		library.cache[path] = &sourceFile{Path: path, Content: []byte(raw), Lines: lines}
	}
	asset, err := library.compilePart("test.dat", map[int]colorValue{4: {Value: "#1b2a34", Edge: "#333333"}})
	if err != nil {
		t.Fatal(err)
	}
	if asset.Bounds != [6]float64{8, 20, 30, 10, 22, 30} || asset.TriangleCount != 1 {
		t.Fatalf("render controls changed mesh bounds: %v, %d triangles", asset.Bounds, asset.TriangleCount)
	}
	raw, err := base64.StdEncoding.DecodeString(asset.Data)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Nodes []struct {
			Extras struct {
				Lines renderLineData `json:"ldrawLines"`
			} `json:"extras"`
		} `json:"nodes"`
		Materials []struct {
			PBR struct {
				Color []float64 `json:"baseColorFactor"`
			} `json:"pbrMetallicRoughness"`
		} `json:"materials"`
	}
	if err := json.Unmarshal(raw[20:20+binary.LittleEndian.Uint32(raw[12:16])], &doc); err != nil {
		t.Fatal(err)
	}
	data := doc.Nodes[0].Extras.Lines
	if data.Version != 1 || len(data.Groups) != 3 {
		t.Fatalf("missing line groups: %#v", data)
	}
	found := false
	for _, group := range data.Groups {
		if group.Conditional {
			found = true
			want := []float64{10, 20, 30, 8, 20, 30, 10, 10020, 30, 8, 10020, 30}
			if group.Color != "#333333" || !reflect.DeepEqual(group.Vertices, want) {
				t.Fatalf("conditional line: %#v", group)
			}
		}
	}
	if !found {
		t.Fatal("conditional line was lost")
	}
	if math.Abs(doc.Materials[0].PBR.Color[0]-0.010960094006488246) > 1e-10 {
		t.Fatalf("fixed color is not linear: %v", doc.Materials[0].PBR.Color)
	}
	again, err := library.compilePart("test.dat", map[int]colorValue{4: {Value: "#1b2a34", Edge: "#333333"}})
	if err != nil || again.Hash != asset.Hash {
		t.Fatalf("non-deterministic output: %v", err)
	}
}

func TestRenderLinesKeepCurrentColorInheritance(t *testing.T) {
	parent := materialContext{Surface: "current", Edge: "edge-current"}
	groups := groupRenderLines([]renderLine{
		{Points: []vec3{{}, {X: 1}}, Material: geometryMaterial(24, childMaterial(16, parent))},
		{Points: []vec3{{}, {X: 1}}, Material: geometryMaterial(16, childMaterial(16, parent))},
	}, nil)
	if groups.Groups[0].Color != "current" || groups.Groups[1].Color != "edge-current" {
		t.Fatalf("lost inherited colors: %#v", groups)
	}
	if _, err := parseSource([]byte("5 24 0 0 0 1 0 0\n")); err == nil {
		t.Fatal("accepted incomplete conditional line")
	}
}

// Optional real-library acceptance; default tests remain self-contained.
func TestEmbeddedCatalogMatchesServerCompiler(t *testing.T) {
	archive := os.Getenv("CHIMII_LDRAW_ARCHIVE")
	if archive == "" {
		t.Skip("set CHIMII_LDRAW_ARCHIVE to the pinned official archive")
	}
	lock, err := ParseLock(embeddedCatalogLock)
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
	source, err := os.ReadFile("../../../packages/views/build/catalog/catalog.generated.ts")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), `LDRAW_CATALOG_VERSION = "`+composeCatalogVersion(lock.Release, lock.ArchiveSHA256)+`"`) {
		t.Fatal("embedded/server catalog versions differ")
	}
	parts, err := library.CompileParts(lock.RootParts, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range parts {
		if !strings.Contains(string(source), `hash: "`+part.ContentSHA256+`"`) {
			t.Errorf("embedded/server GLBs differ for %s", part.PartID)
		}
	}
}
