package handler

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	buildstudio "github.com/chimii-ai/chimii/server/internal/build"
	db "github.com/chimii-ai/chimii/server/pkg/db/generated"
	"github.com/chimii-ai/chimii/server/pkg/llm"
)

func (h *Handler) buildPlanningEffort() string {
	if v := strings.TrimSpace(h.cfg.BuildPlannerEffort); v != "" {
		return v
	}
	return "low"
}
func (h *Handler) buildReviewEffort() string {
	if v := strings.TrimSpace(h.cfg.BuildReviewEffort); v != "" {
		return v
	}
	return "low"
}

func restoreSceneVisual(recipe buildstudio.AssemblyRecipe, plan *buildstudio.BuildPlan) bool {
	if recipe.Scene.Version != 2 {
		return true
	}
	var visual buildstudio.VisualReviewEvidence
	if json.Unmarshal([]byte(recipe.Metadata["scene_review_visual"]), &visual) != nil || visual.Renderer != buildstudio.SceneRendererVersion || visual.ContentHash != plan.ContentHash || len(visual.Views) != 6 {
		return false
	}
	for i, name := range []string{"front", "back", "left", "right", "top", "isometric"} {
		v := visual.Views[i]
		hash, err := hex.DecodeString(v.SHA256)
		if err != nil || len(hash) != 32 || v.Direction != name {
			return false
		}
	}
	plan.Validation.Evidence.Visual = &visual
	return true
}

func (h *Handler) sceneReviewImages(ctx context.Context, plan buildstudio.BuildPlan) ([]llm.Image, *buildstudio.VisualReviewEvidence, error) {
	var views []buildstudio.SceneView
	var err error
	if h.cfg.BuildSceneViewRenderer != nil {
		views, err = h.cfg.BuildSceneViewRenderer(ctx, plan)
	} else {
		if h.Queries == nil {
			return nil, nil, fmt.Errorf("catalog-backed review renderer is not configured")
		}
		renderCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		palette := map[int]uint32{}
		for _, c := range buildCatalogColors() {
			value, e := strconv.ParseUint(strings.TrimPrefix(c.Hex, "#"), 16, 32)
			if e != nil {
				return nil, nil, e
			}
			palette[c.Code] = uint32(value)
		}
		views, err = buildstudio.RenderSceneViews(renderCtx, plan, func(ctx context.Context, version, id string) ([]byte, error) {
			asset, e := h.Queries.GetLDrawPartRevisionByVersionAndPartID(ctx, db.GetLDrawPartRevisionByVersionAndPartIDParams{CatalogVersion: version, PartID: id})
			if e != nil {
				return nil, e
			}
			return asset.Payload, nil
		}, palette)
	}
	if err != nil {
		return nil, nil, err
	}
	if len(views) != 6 {
		return nil, nil, fmt.Errorf("review requires six rendered catalog views")
	}
	images := make([]llm.Image, 0, len(views))
	for _, view := range views {
		if len(view.PNG) == 0 {
			return nil, nil, fmt.Errorf("empty rendered review view")
		}
		images = append(images, llm.Image{MediaType: "image/png", Data: view.PNG})
	}
	return images, &buildstudio.VisualReviewEvidence{Renderer: buildstudio.SceneRendererVersion, ContentHash: plan.ContentHash, Views: views}, nil
}
