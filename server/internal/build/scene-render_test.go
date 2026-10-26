package build

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func reviewTestAssets(t *testing.T) func(context.Context, string, string) ([]byte, error) {
	t.Helper()
	raw, err := os.ReadFile("../../../packages/views/build/catalog/catalog.generated.ts")
	if err != nil {
		t.Fatal(err)
	}
	assets := map[string][]byte{}
	pattern := regexp.MustCompile(`(?s)"([a-z0-9]+\.dat)": \{.*?glbBase64: "([^"]+)"`)
	for _, m := range pattern.FindAllSubmatch(raw, -1) {
		data, err := base64.StdEncoding.DecodeString(string(m[2]))
		if err != nil {
			t.Fatal(err)
		}
		assets[string(m[1])] = data
	}
	if len(assets) != 10 {
		t.Fatalf("embedded real catalog assets=%d", len(assets))
	}
	return func(_ context.Context, version, id string) ([]byte, error) {
		if version != CatalogVersion {
			return nil, fmt.Errorf("wrong catalog revision")
		}
		data, ok := assets[id]
		if !ok {
			return nil, fmt.Errorf("asset absent")
		}
		return data, nil
	}
}

func TestSceneViewsUseRealGeometryColorsAndDeterministicCameras(t *testing.T) {
	plan := BuildPlan{CatalogVersion: CatalogVersion, Parts: StarterCatalog, Placements: []Placement{
		{ID: "base", PartID: "brick-2x4", Color: 4},
		{ID: "top", PartID: "brick-2x2", Color: 1, X: 1, Y: 3},
	}}
	palette := map[int]uint32{4: 0xb40000, 1: 0x1e5aa8}
	loader := reviewTestAssets(t)
	views, err := RenderSceneViews(context.Background(), plan, loader, palette)
	if err != nil {
		t.Fatal(err)
	}
	if len(views) != 6 {
		t.Fatal("missing required views")
	}
	again, err := RenderSceneViews(context.Background(), plan, loader, palette)
	if err != nil {
		t.Fatal(err)
	}
	for i, view := range views {
		if view.SHA256 != again[i].SHA256 {
			t.Fatal("non-deterministic image")
		}
		image, err := png.Decode(bytes.NewReader(view.PNG))
		if err != nil || image.Bounds().Dx() != 336 {
			t.Fatalf("invalid PNG: %v", err)
		}
		red, blue := 0, 0
		for y := 0; y < 336; y++ {
			for x := 0; x < 336; x++ {
				r, g, b, _ := image.At(x, y).RGBA()
				if r > 2*g && r > 2*b {
					red++
				}
				if b > 2*r && b > g {
					blue++
				}
			}
		}
		if view.Direction == "front" && (red < 20 || blue < 20) {
			t.Fatalf("placement colors or occlusion lost: red=%d blue=%d", red, blue)
		}
		if out := os.Getenv("CHIMII_SCENE_RENDER_OUTPUT"); out != "" {
			if err := os.MkdirAll(out, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(out, view.Direction+".png"), view.PNG, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = RenderSceneViews(ctx, plan, loader, palette); err == nil {
		t.Fatal("cancelled render succeeded")
	}
	if _, err = RenderSceneViews(context.Background(), plan, func(context.Context, string, string) ([]byte, error) { return nil, fmt.Errorf("offline") }, palette); err == nil {
		t.Fatal("missing real geometry was substituted")
	}
}

func TestSceneGLBDecoderRejectsMalformedAndUnsupportedAssets(t *testing.T) {
	for _, raw := range [][]byte{nil, []byte("not GLB"), make([]byte, 40)} {
		if _, err := decodeRenderPart(raw); err == nil {
			t.Fatal("malformed GLB accepted")
		}
	}
	loader := reviewTestAssets(t)
	raw, _ := loader(context.Background(), CatalogVersion, "3001.dat")
	raw = append([]byte{}, raw...)
	raw[12] = 255
	raw[13] = 255
	raw[14] = 255
	raw[15] = 127
	if _, err := decodeRenderPart(raw); err == nil {
		t.Fatal("invalid chunk range accepted")
	}
}

func TestSceneViewsRespectYawAndDepth(t *testing.T) {
	loader := reviewTestAssets(t)
	palette := map[int]uint32{4: 0xb40000, 1: 0x1e5aa8}
	plan := BuildPlan{CatalogVersion: CatalogVersion, Parts: StarterCatalog, Placements: []Placement{{ID: "single", PartID: "brick-2x4", Color: 4, X: -3, Z: -2}}}
	widths := []int{}
	for _, rotation := range []int{0, 90, 180, 270} {
		plan.Placements[0].Rotation = rotation
		views, err := RenderSceneViews(context.Background(), plan, loader, palette)
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(views[0].PNG))
		if err != nil {
			t.Fatal(err)
		}
		left, right := 336, -1
		for y := 0; y < 336; y++ {
			for x := 0; x < 336; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				if r > 2*g && r > 2*b {
					left = min(left, x)
					right = max(right, x)
				}
			}
		}
		widths = append(widths, right-left+1)
	}
	if widths[0] < widths[1]*3/2 || widths[0] != widths[2] || widths[1] != widths[3] {
		t.Fatalf("yaw footprint mismatch: %v", widths)
	}
	plan.Placements = []Placement{{ID: "front", PartID: "brick-2x4", Color: 4, Z: 0}, {ID: "back", PartID: "brick-2x4", Color: 1, Z: 4}}
	views, err := RenderSceneViews(context.Background(), plan, loader, palette)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		img, err := png.Decode(bytes.NewReader(views[i].PNG))
		if err != nil {
			t.Fatal(err)
		}
		red, blue := 0, 0
		for y := 0; y < 336; y++ {
			for x := 0; x < 336; x++ {
				r, g, b, _ := img.At(x, y).RGBA()
				if r > 2*g && r > 2*b {
					red++
				}
				if b > 2*r && b > g {
					blue++
				}
			}
		}
		if i == 0 && (red == 0 || blue != 0) || i == 1 && (blue == 0 || red != 0) {
			t.Fatalf("depth buffer reversed %s: red=%d blue=%d", views[i].Direction, red, blue)
		}
	}
}
