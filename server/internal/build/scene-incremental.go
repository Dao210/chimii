package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// SceneCompileCache belongs to one bounded generation/review execution. It must
// not be shared across requests or persisted as mutable creation state.
type SceneCompileCache struct {
	prefixes map[string][]Placement
	order    []string
	Hits     int
}

func NewSceneCompileCache() *SceneCompileCache {
	return &SceneCompileCache{prefixes: map[string][]Placement{}}
}

func (c *SceneCompileCache) remember(key string, parts []Placement) {
	if c == nil {
		return
	}
	if c.prefixes == nil {
		c.prefixes = map[string][]Placement{}
	}
	if _, exists := c.prefixes[key]; !exists {
		if len(c.order) >= 128 {
			delete(c.prefixes, c.order[0])
			c.order = c.order[1:]
		}
		c.order = append(c.order, key)
	}
	c.prefixes[key] = append([]Placement{}, parts...)
}

func scenePrefixKey(catalogHash string, target designTarget, fixed []Placement) string {
	type cell struct {
		Position DesignVector
		Value    targetCell
	}
	cells := make([]cell, 0, len(target))
	for _, p := range sortedTargetCells(target) {
		cells = append(cells, cell{p, target[p]})
	}
	raw, _ := json.Marshal(struct {
		Version string
		Catalog string
		Cells   []cell
		Fixed   []Placement
	}{SceneGeneratorVersion, catalogHash, cells, fixed})
	hash := sha256.Sum256(raw)
	return hex.EncodeToString(hash[:])
}

func compileSceneObject(ctx context.Context, object SceneObject, target designTarget, fixed []Placement, parts []PartSpec, catalog PartCatalog, budget *searchBudget, maxParts int, cache *SceneCompileCache) ([]Placement, int, error) {
	if len(target) == 0 {
		return append([]Placement{}, fixed...), 0, nil
	}
	ordered := sortedTargetCells(target)
	bottom := ordered[0].Y
	frontiers := map[int]designTarget{}
	for p, cell := range target {
		level := bottom + ((p.Y-bottom)/12+1)*12
		if frontiers[level] == nil {
			frontiers[level] = designTarget{}
		}
		frontiers[level][p] = cell
	}
	levels := make([]int, 0, len(frontiers))
	for level := range frontiers {
		levels = append(levels, level)
	}
	sort.Ints(levels)
	raw, _ := json.Marshal(catalog)
	digest := sha256.Sum256(raw)
	catalogHash := hex.EncodeToString(digest[:])
	prefix := designTarget{}
	compiled := append([]Placement{}, fixed...)
	for stage, level := range levels {
		if err := ctx.Err(); err != nil {
			return nil, stage, err
		}
		band := frontiers[level]
		for p, v := range band {
			prefix[p] = v
		}
		key := scenePrefixKey(catalogHash, prefix, fixed)
		if cache != nil {
			if saved, ok := cache.prefixes[key]; ok {
				compiled = append([]Placement{}, saved...)
				cache.Hits++
				continue
			}
		}
		// Freeze bonded components, not independent stacks inside one continuous
		// foundation. Separate grounded legs may wait for a later connecting torso.
		allowSeparate := stage < len(levels)-1 && checkSceneTargetConnected(prefix, fixed, catalog) != nil
		validate := func(p []Placement) ValidationReport {
			r := validateAssembly(p, deriveExactConnections(p, catalog), UnlimitedInventory(), catalog, -64, 64)
			if allowSeparate {
				issues := []ValidationIssue{}
				for _, issue := range r.Issues {
					if issue.Code != "disconnected" {
						issues = append(issues, issue)
					}
				}
				r.Issues = issues
				r.Buildable = len(issues) == 0
			}
			return r
		}
		var found []Placement
		var dead *DesignVector
		for variant := 0; variant < 6 && !budget.exhausted(); variant++ {
			s := &shapeSearch{target: band, cells: sortedTargetCells(band), parts: parts, catalog: catalog, inventory: UnlimitedInventory(), remaining: map[inventoryKey]int{}, occupied: map[DesignVector]int{}, budget: budget, start: budget.used, limit: min(16000, budget.limit-budget.used), variant: variant, seamPriority: variant > 0, maxParts: min(maxObjectParts, maxParts), validate: validate, normalize: func(p []Placement) []Placement { return sceneSteps(p, catalog) }, frozenIDs: map[string]bool{}}
			for _, p := range compiled {
				index := len(s.placements)
				s.placements = append(s.placements, p)
				s.frozenIDs[p.ID] = true
				size := orientedSizeWith(p, catalog)
				for x := 0; x < size.x; x++ {
					for y := 0; y < catalog[p.PartID].PlatesY; y++ {
						for z := 0; z < size.z; z++ {
							q := DesignVector{p.X + x, p.Y + y, p.Z + z}
							if _, overlap := band[q]; overlap {
								return nil, stage, designError("incremental target overlaps fixed part: " + p.ID)
							}
							s.occupied[q] = index
							s.seedCells++
						}
					}
				}
			}
			if s.walk(0) {
				found = s.result
				break
			}
			if s.deadCell != nil {
				q := *s.deadCell
				dead = &q
			}
		}
		if len(found) == 0 {
			code := BuildErrorStructureInvalid
			if budget.exhausted() || ctx.Err() != nil {
				code = BuildErrorSearchLimit
			}
			where := ""
			if dead != nil {
				where = fmt.Sprintf("; construction %s at (%d,%d,%d)", band[*dead].Node, dead.X, dead.Y, dead.Z)
			}
			return nil, stage, &BuildError{Code: code, Cause: fmt.Errorf("object %s frontier below Y=%d has no verified layout after %d shared nodes; %d lower stages retained%s", object.ID, level, budget.used, stage, where)}
		}
		compiled = found
		cache.remember(key, compiled)
	}
	return compiled, len(levels), nil
}
