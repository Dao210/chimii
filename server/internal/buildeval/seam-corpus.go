package buildeval

import (
	"fmt"

	"github.com/chimii-ai/chimii/server/internal/build"
)

const SeamCorpusVersion = "build-seams-v1"

// SeamCases probes catalog tiling with exact counts and reusable inventory.
// The 80 fixture rows are frozen from random rectangular tilings (seed 42),
// including every failure. They are synthetic compiler inputs, not LLM output
// or the StableText2Brick dataset. Columns are width, brick layers, depth,
// followed by quantities of 1x1, 1x2, 2x2 and 2x4 bricks.
var seamFixtures = [][7]int{
	{4, 2, 5, 12, 6, 2, 1},
	{7, 3, 3, 19, 12, 5, 0},
	{6, 3, 5, 20, 11, 4, 4},
	{6, 2, 4, 8, 6, 5, 1},
	{5, 3, 3, 15, 7, 4, 0},
	{5, 2, 2, 4, 4, 2, 0},
	{8, 3, 3, 24, 12, 6, 0},
	{5, 3, 5, 15, 10, 6, 2},
	{6, 2, 4, 8, 2, 1, 4},
	{8, 2, 4, 10, 13, 3, 2},
	{8, 2, 4, 12, 8, 3, 3},
	{5, 3, 3, 21, 2, 1, 2},
	{4, 3, 3, 8, 8, 3, 0},
	{4, 3, 2, 10, 5, 1, 0},
	{7, 3, 2, 8, 5, 4, 1},
	{7, 3, 2, 6, 6, 2, 2},
	{8, 2, 2, 6, 3, 5, 0},
	{8, 3, 3, 18, 13, 5, 1},
	{4, 2, 3, 6, 5, 2, 0},
	{5, 2, 5, 12, 5, 3, 2},
	{7, 3, 4, 18, 11, 3, 4},
	{6, 3, 4, 22, 9, 2, 3},
	{4, 2, 4, 8, 2, 1, 2},
	{8, 2, 4, 18, 3, 4, 3},
	{6, 3, 2, 8, 4, 5, 0},
	{7, 2, 4, 8, 4, 6, 2},
	{4, 3, 5, 12, 8, 4, 2},
	{5, 2, 5, 12, 9, 5, 0},
	{7, 3, 5, 21, 16, 7, 3},
	{8, 3, 4, 14, 5, 10, 4},
	{4, 2, 4, 8, 6, 1, 1},
	{4, 3, 4, 14, 9, 4, 0},
	{4, 3, 4, 12, 12, 3, 0},
	{8, 2, 3, 18, 7, 0, 2},
	{6, 3, 5, 16, 15, 5, 3},
	{6, 2, 3, 12, 6, 1, 1},
	{4, 3, 4, 22, 9, 2, 0},
	{7, 2, 2, 2, 5, 0, 2},
	{5, 2, 5, 14, 6, 6, 0},
	{5, 3, 5, 13, 9, 5, 3},
	{8, 2, 2, 12, 0, 5, 0},
	{6, 2, 5, 10, 7, 7, 1},
	{4, 3, 2, 10, 3, 2, 0},
	{6, 2, 2, 4, 2, 2, 1},
	{8, 2, 4, 14, 9, 4, 2},
	{8, 2, 4, 14, 9, 6, 1},
	{6, 2, 5, 6, 9, 7, 1},
	{7, 2, 5, 12, 9, 8, 1},
	{7, 2, 2, 8, 4, 1, 1},
	{7, 2, 5, 20, 7, 9, 0},
	{5, 2, 5, 8, 7, 3, 2},
	{4, 3, 4, 14, 5, 2, 2},
	{5, 3, 4, 16, 12, 3, 1},
	{6, 3, 4, 14, 11, 5, 2},
	{8, 3, 4, 20, 6, 6, 5},
	{5, 3, 5, 21, 7, 4, 3},
	{5, 3, 5, 21, 7, 6, 2},
	{6, 3, 5, 26, 10, 7, 2},
	{6, 2, 4, 12, 6, 4, 1},
	{6, 2, 3, 16, 6, 2, 0},
	{6, 3, 5, 16, 15, 7, 2},
	{4, 3, 2, 8, 6, 1, 0},
	{7, 2, 3, 16, 3, 3, 1},
	{7, 3, 4, 26, 11, 5, 2},
	{7, 2, 3, 10, 4, 6, 0},
	{5, 3, 4, 14, 5, 9, 0},
	{7, 2, 2, 10, 3, 1, 1},
	{6, 2, 4, 10, 5, 5, 1},
	{7, 2, 4, 14, 7, 7, 0},
	{8, 2, 3, 12, 6, 4, 1},
	{5, 2, 5, 16, 9, 2, 1},
	{8, 2, 3, 14, 5, 4, 1},
	{6, 2, 4, 6, 5, 2, 3},
	{6, 3, 5, 16, 9, 6, 4},
	{6, 3, 5, 18, 14, 9, 1},
	{5, 3, 4, 24, 8, 5, 0},
	{5, 2, 3, 10, 4, 3, 0},
	{8, 2, 3, 10, 7, 2, 2},
	{5, 2, 3, 6, 4, 4, 0},
	{5, 2, 3, 16, 3, 2, 0},
}

func SeamCases() []Case {
	cases := make([]Case, 0, len(seamFixtures))
	for n, row := range seamFixtures {
		parts := []string{"brick-1x1", "brick-1x2", "brick-2x2", "brick-2x4"}
		items := []build.InventoryItem{}
		count := 0
		for i, id := range parts {
			if row[i+3] > 0 {
				items = append(items, build.InventoryItem{PartID: id, Color: 1, Quantity: row[i+3]})
				count += row[i+3]
			}
		}
		r := recipe(box("solid", 0, 0, 0, row[0], row[1]*3, row[2]))
		r.Constraints.PartCount = count
		cases = append(cases, Case{ID: fmt.Sprintf("seam-%03d", n+1), Family: "inventory-seams",
			Prompt: "保持长方体体积、颜色和精确块数，用现有零件连接整个作品", Expectation: "observe", Recipe: r,
			Inventory: build.NewInventorySnapshot(true, 1, items)})
	}
	return cases
}
