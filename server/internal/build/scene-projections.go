package build

import (
	"fmt"
	"sort"
	"strings"
)

type SceneProjection struct {
	Object          string         `json:"object"`
	View            string         `json:"view"`
	HorizontalUnit  string         `json:"horizontal_unit"`
	VerticalUnit    string         `json:"vertical_unit"`
	HorizontalScale int            `json:"horizontal_scale"`
	VerticalScale   int            `json:"vertical_scale"`
	Rows            []string       `json:"rows"`
	Legend          map[string]int `json:"legend"`
}

// SceneProjections emits bounded, deterministic occupancy projections for
// semantic review. They do not imply material collision or photorealistic review.
func SceneProjections(plan BuildPlan) []SceneProjection {
	groups := map[string][]Placement{}
	for _, p := range plan.Placements {
		groups[p.Object] = append(groups[p.Object], p)
	}
	ids := []string{}
	for id := range groups {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	views := []SceneProjection{}
	for _, id := range ids {
		cells := map[DesignVector]int{}
		for _, p := range groups[id] {
			s := orientedSizeWith(p, plan.Parts)
			for x := 0; x < s.x; x++ {
				for z := 0; z < s.z; z++ {
					for y := 0; y < plan.Parts[p.PartID].PlatesY; y++ {
						cells[DesignVector{p.X + x, p.Y + y, p.Z + z}] = p.Color
					}
				}
			}
		}
		for _, view := range []string{"front", "right", "top"} {
			minU, minV, maxU, maxV := 1000, 1000, -1000, -1000
			type pixel struct{ color, depth int }
			pixels := map[[2]int]pixel{}
			for p, color := range cells {
				u, v, depth := p.X, p.Y, -p.Z
				if view == "right" {
					u, depth = p.Z, p.X
				}
				if view == "top" {
					v, depth = p.Z, p.Y
				}
				minU, minV, maxU, maxV = min(minU, u), min(minV, v), max(maxU, u), max(maxV, v)
				key := [2]int{u, v}
				if old, ok := pixels[key]; !ok || depth > old.depth {
					pixels[key] = pixel{color, depth}
				}
			}
			if len(pixels) == 0 {
				continue
			}
			hu := max(1, (maxU-minU+40)/40)
			vu := 3
			if view == "top" {
				vu = hu
			}
			vu = max(vu, (maxV-minV+31)/32)
			width, height := (maxU-minU)/hu+1, (maxV-minV)/vu+1
			grid := make([][]byte, height)
			for i := range grid {
				grid[i] = []byte(strings.Repeat(".", width))
			}
			legend := map[string]int{}
			codes := AllowedColorCodes()
			symbols := map[int]byte{}
			for n, c := range codes {
				symbol := byte('A' + n)
				symbols[c] = symbol
				legend[fmt.Sprintf("%c", symbol)] = c
			}
			// Sample in fixed coordinate order so downsampling never depends on Go map iteration.
			for v := minV; v <= maxV; v++ {
				for u := minU; u <= maxU; u++ {
					if p, ok := pixels[[2]int{u, v}]; ok {
						grid[height-1-(v-minV)/vu][(u-minU)/hu] = symbols[p.color]
					}
				}
			}
			rows := []string{}
			for _, row := range grid {
				rows = append(rows, string(row))
			}
			vertical := "plates"
			if view == "top" {
				vertical = "studs"
			}
			views = append(views, SceneProjection{Object: id, View: view, HorizontalUnit: "studs", VerticalUnit: vertical, HorizontalScale: hu, VerticalScale: vu, Rows: rows, Legend: legend})
		}
	}
	return views
}
