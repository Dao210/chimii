// Package buildeval provides offline experiments; it never publishes creations.
package buildeval

import (
	"fmt"

	"github.com/chimii-ai/chimii/server/internal/build"
)

const CorpusVersion = "build-baseline-v1"

type Case struct {
	ID              string                  `json:"id"`
	Family          string                  `json:"family"`
	Prompt          string                  `json:"prompt"`
	Expectation     string                  `json:"expectation"`
	Recipe          *build.AssemblyRecipe   `json:"recipe,omitempty"`
	Placements      []build.Placement       `json:"placements,omitempty"`
	Inventory       build.InventorySnapshot `json:"inventory"`
	SourceCandidate *Candidate              `json:"source_candidate,omitempty"`
	PreflightError  string                  `json:"preflight_error,omitempty"`
	// Edits must preserve the named source shapes byte-for-byte.
	Source      *build.DesignSpec `json:"source,omitempty"`
	PreserveIDs []string          `json:"preserve_ids,omitempty"`
}

func box(id string, x, y, z, sx, sy, sz int) build.ShapeNode {
	return build.ShapeNode{ID: id, Label: id, Kind: "box", Operation: "add", Position: build.DesignVector{X: x, Y: y, Z: z}, Size: build.DesignVector{X: sx, Y: sy, Z: sz}, Color: 1}
}

func recipe(shapes ...build.ShapeNode) *build.AssemblyRecipe {
	return &build.AssemblyRecipe{Version: 3, Subject: "offline-evaluation", Title: "Offline evaluation", Constraints: build.RecipeConstraints{ExactColors: true}, Design: &build.DesignSpec{Version: 1, Mode: "static", Shapes: shapes}}
}

func placed(id, part string, x, y, z, step int) build.Placement {
	return build.Placement{ID: id, PartID: part, X: x, Y: y, Z: z, Step: step, Color: 1, Module: "probe"}
}

// BaselineCases contains 100 deterministic parametric cases, not 100 human
// studies or live language-model calls. Observe cases have no safety oracle.
func BaselineCases() []Case {
	cases := make([]Case, 0, 100)
	add := func(family string, n int, prompt, expectation string, r *build.AssemblyRecipe, p []build.Placement, inv build.InventorySnapshot) {
		cases = append(cases, Case{ID: fmt.Sprintf("%s-%02d", family, n+1), Family: family, Prompt: prompt, Expectation: expectation, Recipe: r, Placements: p, Inventory: inv})
	}
	u := build.UnlimitedInventory()
	for n := 0; n < 10; n++ {
		// Height, footprint and the inventory shortage vary independently across families.
		add("tower", n, fmt.Sprintf("搭一座 %d 层方塔", n+2), "accept", recipe(box("tower", 0, 0, 0, 2, 3*(n+2), 2)), nil, u)
		add("thin-wall", n, fmt.Sprintf("搭一面长 %d 格的双层薄墙", n+2), "accept", recipe(box("wall", 0, 0, 0, n+2, 6, 1)), nil, u)
		outer := box("frame", -4, 0, -4, 8, 6, 8)
		hole := box("hole", -2, 0, -2, 2+n%3, 6, 2+n/3)
		hole.Operation = "subtract"
		add("hole", n, "搭一个保留贯穿孔洞的方框", "observe", recipe(outer, hole), nil, u)
		// Disconnected bottom pieces must be joined through the next layer.
		span := 4 + n
		add("bridge", n, fmt.Sprintf("搭一座跨度 %d 格的双脚桥", span), "observe", recipe(box("left", 0, 0, 0, 2, 3, 2), box("right", span-2, 0, 0, 2, 3, 2), box("beam", 0, 3, 0, span, 6, 2)), nil, u)
		count := n + 2
		exact := recipe(box("stack", 0, 0, 0, 4, count*3, 2))
		exact.Constraints.PartCount = count
		inv := build.NewInventorySnapshot(true, 1, []build.InventoryItem{{PartID: "brick-2x4", Color: 1, Quantity: count}})
		add("exact-count", n, fmt.Sprintf("只用 %d 块蓝色 2x4 砖搭塔", count), "accept", exact, nil, inv)
		short := inv
		short.Items = []build.InventoryItem{{PartID: "brick-2x4", Color: 1, Quantity: count - 1}}
		short = build.NewInventorySnapshot(true, 1, short.Items)
		add("missing-parts", n, "保持块数要求，库存少一块时不能通过", build.BuildErrorInsufficientInventory, exact, nil, short)
		wrongColor := build.NewInventorySnapshot(true, 1, []build.InventoryItem{{PartID: "brick-2x4", Color: 4, Quantity: count}})
		add("exact-color", n, "要求蓝色，只拥有红色时不能改色通过", build.BuildErrorInsufficientInventory, exact, nil, wrongColor)
		// The opening is unchanged while the wall above it is lowered.
		wall := box("wall", 0, 0, 0, 6, 12+3*n, 2)
		door := box("door", 2, 0, 0, 2, 6, 2)
		door.Operation = "subtract"
		source := recipe(wall, door).Design
		wall.Size.Y -= 3
		add("edit-opening", n, "降低门墙，保留门洞的位置和尺寸", "observe", recipe(wall, door), nil, u)
		cases[len(cases)-1].Source, cases[len(cases)-1].PreserveIDs = source, []string{"door"}
		// Staircase overhang probes separate ground tipping from stud attachment.
		p := []build.Placement{placed("base", "brick-2x4", 0, 0, 0, 1)}
		for level := 1; level <= n+1; level++ {
			p = append(p, placed(fmt.Sprintf("overhang-%d", level), "brick-2x4", level, level*3, 0, level+1))
		}
		add("cantilever", n, "逐层向外悬伸，比较支撑和连接受力", "observe", nil, p, u)
		invalid := []build.Placement{placed("a", "brick-2x2", 0, 0, 0, 1), placed("b", "brick-2x2", 0, 0, 0, 2)}
		switch n % 5 {
		case 1:
			invalid[1].Y = 9 // floating
		case 2:
			invalid[1].X = 5 // disconnected
		case 3:
			invalid[1].PartID = "invented-part"
		case 4:
			invalid[1].Rotation = 45
		}
		invalid[0].X, invalid[1].X = invalid[0].X-n/5, invalid[1].X-n/5
		add("invalid-layout", n, "重叠、悬空、断开、未知零件或非法角度必须拒绝", "reject", nil, invalid, u)
	}
	return cases
}
