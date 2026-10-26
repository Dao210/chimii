package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
)

// Fixture transport tests still render the project's actual pinned GLB assets.
func sceneReviewFixtureRenderer(ctx context.Context, plan buildstudio.BuildPlan) ([]buildstudio.SceneView, error) {
	raw, err := os.ReadFile("../../../packages/views/build/catalog/catalog.generated.ts")
	if err != nil {
		return nil, err
	}
	assets := map[string][]byte{}
	for _, m := range regexp.MustCompile(`(?s)"([a-z0-9]+\.dat)": \{.*?glbBase64: "([^"]+)"`).FindAllSubmatch(raw, -1) {
		data, err := base64.StdEncoding.DecodeString(string(m[2]))
		if err != nil {
			return nil, err
		}
		assets[string(m[1])] = data
	}
	palette := map[int]uint32{}
	for _, c := range buildCatalogColors() {
		rgb, err := strconv.ParseUint(strings.TrimPrefix(c.Hex, "#"), 16, 32)
		if err != nil {
			return nil, err
		}
		palette[c.Code] = uint32(rgb)
	}
	return buildstudio.RenderSceneViews(ctx, plan, func(_ context.Context, version, id string) ([]byte, error) {
		if version != buildstudio.CatalogVersion {
			return nil, fmt.Errorf("fixture has no assets for %s", version)
		}
		asset, ok := assets[id]
		if !ok {
			return nil, fmt.Errorf("fixture has no geometry for %s", id)
		}
		return asset, nil
	}, palette)
}

func TestBuildSceneVisualReviewUsesImagesAndInvalidatesOldRenderer(t *testing.T) {
	calls := 0
	h := &Handler{LLM: sceneTestLLM(func(r *http.Request) (string, error) {
		calls++
		data, _ := io.ReadAll(r.Body)
		var request struct {
			Messages []struct {
				Content []struct {
					Type   string
					Source struct {
						MediaType string `json:"media_type"`
						Data      string
					}
				}
			}
			Output struct{ Effort string } `json:"output_config"`
		}
		if err := json.Unmarshal(data, &request); err != nil {
			t.Fatal(err)
		}
		images := 0
		for _, message := range request.Messages {
			for _, block := range message.Content {
				if block.Type != "image" {
					continue
				}
				images++
				png, err := base64.StdEncoding.DecodeString(block.Source.Data)
				if err != nil || len(png) < 8 || string(png[:8]) != "\x89PNG\r\n\x1a\n" || block.Source.MediaType != "image/png" {
					t.Fatal("review received no real PNG")
				}
			}
		}
		if images != 6 || request.Output.Effort != "low" {
			t.Fatalf("images=%d effort=%s", images, request.Output.Effort)
		}
		return `{"outcome":"approved","message":"Fixture image review."}`, nil
	})}
	h.cfg.BuildSceneViewRenderer = sceneReviewFixtureRenderer
	save := func(buildstudio.AssemblyRecipe) error { return nil }
	approved, err := h.compileAndReviewScene(context.Background(), scenePlanningFixture(), buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, save)
	if err != nil || calls != 1 || approved.Plan.Validation.Evidence.Visual == nil {
		t.Fatalf("image review: calls=%d err=%v", calls, err)
	}
	for _, change := range []func(map[string]string){
		func(m map[string]string) { delete(m, "scene_review_visual") },
		func(m map[string]string) {
			m["scene_review_visual"] = strings.ReplaceAll(m["scene_review_visual"], buildstudio.SceneRendererVersion, "obsolete-renderer")
		},
		func(m map[string]string) { m["scene_review_visual"] = `{"renderer":"bad"}` },
	} {
		r := approved.Recipe
		r.Metadata = maps.Clone(r.Metadata)
		change(r.Metadata)
		before := calls
		result, err := h.compileAndReviewScene(context.Background(), r, buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, save)
		if err != nil || calls != before+1 || result.Plan.Validation.Evidence.Visual.Renderer != buildstudio.SceneRendererVersion {
			t.Fatalf("obsolete review was accepted: %v", err)
		}
	}
	before := calls
	h.cfg.BuildSceneViewRenderer = func(context.Context, buildstudio.BuildPlan) ([]buildstudio.SceneView, error) {
		return nil, fmt.Errorf("missing pinned GLB")
	}
	if _, err := h.compileAndReviewScene(context.Background(), scenePlanningFixture(), buildstudio.UnlimitedInventory(), buildstudio.StarterCatalog, save); err == nil || calls != before {
		t.Fatal("missing geometry fell back to text approval")
	}
}

// Review images and the delivered GLB viewer must use the same current-color
// palette, otherwise a reviewer approves different colors from the user.
func TestBuildScenePaletteMatchesDeliveredViewer(t *testing.T) {
	raw, err := os.ReadFile("../../../packages/views/build/components/ldraw-render-preset.ts")
	if err != nil {
		t.Fatal(err)
	}
	viewer := map[int]string{}
	for _, m := range regexp.MustCompile(`(?m)^\s*(\d+): 0x([a-f0-9]+),`).FindAllSubmatch(raw, -1) {
		code, _ := strconv.Atoi(string(m[1]))
		viewer[code] = "#" + string(m[2])
	}
	if len(viewer) != len(buildCatalogColors()) {
		t.Fatal("viewer/API color coverage differs")
	}
	for _, c := range buildCatalogColors() {
		if viewer[c.Code] != c.Hex {
			t.Fatalf("color %d differs: viewer=%s review=%s", c.Code, viewer[c.Code], c.Hex)
		}
	}
}
