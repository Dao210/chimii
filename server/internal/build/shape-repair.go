package build

func onlyDisconnectedIssues(validation ValidationReport) bool {
	if len(validation.Issues) == 0 {
		return false
	}
	for _, issue := range validation.Issues {
		if issue.Code != "disconnected" {
			return false
		}
	}
	return true
}

// repairSeams retiles neighboring parts on a disconnected seam. The target
// volume and every feature stay identical; only catalog part boundaries move.
func (s *shapeSearch) repairSeams(input []Placement) ([]Placement, bool) {
	placements := append([]Placement{}, input...)
	for round := 0; round < 48 && s.budget.ctx.Err() == nil; round++ {
		components, count := layoutComponents(placements, s.catalog)
		if count == 1 {
			return placements, ValidateWithCatalog(placements, s.inventory, s.catalog).Buildable
		}
		improved := false
		for i := 0; i < len(placements) && !improved && s.budget.ctx.Err() == nil; i++ {
			for j := i + 1; j < len(placements) && !improved && s.budget.ctx.Err() == nil; j++ {
				if s.budget.used >= s.budget.limit {
					s.limited = true
					return nil, false
				}
				a, b := placements[i], placements[j]
				if components[i] == components[j] || a.Y != b.Y || a.Module != b.Module || a.Color != b.Color || s.catalog[a.PartID].PlatesY != s.catalog[b.PartID].PlatesY {
					continue
				}
				as, bs := orientedSizeWith(a, s.catalog), orientedSizeWith(b, s.catalog)
				touchX := (a.X+as.x == b.X || b.X+bs.x == a.X) && a.Z < b.Z+bs.z && b.Z < a.Z+as.z
				touchZ := (a.Z+as.z == b.Z || b.Z+bs.z == a.Z) && a.X < b.X+bs.x && b.X < a.X+as.x
				if !touchX && !touchZ {
					continue
				}
				if next, ok := s.retileSeamRegion(placements, []int{i, j}, count, round, 160); ok {
					placements, improved = next, true
					continue
				}
				// BrickGPT expands the critical neighborhood before retrying.
				// Include adjacent layers so a locked seam can exchange parts
				// with its neighbors without changing volume or inventory.
				if s.seamPriority {
					region := criticalSeamRegion(placements, s.catalog, i, j)
					if len(region) > 2 {
						if next, ok := s.retileSeamRegion(placements, region, count, round, 640); ok {
							placements, improved = next, true
						}
					}
				}
			}
		}
		if !improved {
			return nil, false
		}
	}
	return nil, false
}

// retileSeamRegion owns a new search state. Failed attempts cannot change the
// caller's placements, input design, or reusable inventory. Every local search
// node consumes the same global budget as the surrounding shape search.
func (s *shapeSearch) retileSeamRegion(placements []Placement, region []int, count, round, limit int) ([]Placement, bool) {
	if s.budget.ctx.Err() != nil || s.budget.used >= s.budget.limit {
		s.limited = true
		return nil, false
	}
	removed := map[int]bool{}
	for _, index := range region {
		removed[index] = true
	}
	s.budget.repairAttempts++
	local := &shapeSearch{budget: s.budget, start: s.budget.used, target: s.target, cells: s.cells, parts: s.parts, catalog: s.catalog, inventory: s.inventory,
		remaining: s.inventory.quantities(), occupied: map[DesignVector]int{}, exactColors: s.exactColors,
		exactCount: s.exactCount, limit: min(limit, s.budget.limit-s.budget.used), variant: round % 3, seamPriority: s.seamPriority}
	for k, p := range placements {
		if s.budget.ctx.Err() != nil {
			s.limited = true
			return nil, false
		}
		if removed[k] {
			continue
		}
		index := len(local.placements)
		local.placements = append(local.placements, p)
		local.remaining[inventoryKey{p.PartID, p.Color}]--
		size := orientedSizeWith(p, s.catalog)
		for x := 0; x < size.x; x++ {
			for z := 0; z < size.z; z++ {
				for y := 0; y < s.catalog[p.PartID].PlatesY; y++ {
					local.occupied[DesignVector{p.X + x, p.Y + y, p.Z + z}] = index
				}
			}
		}
	}
	local.accept = func(candidate []Placement) bool {
		if s.budget.ctx.Err() != nil {
			return false
		}
		_, nextCount := layoutComponents(candidate, s.catalog)
		if nextCount >= count {
			return false
		}
		validation := ValidateWithCatalog(candidate, s.inventory, s.catalog)
		for _, issue := range validation.Issues {
			if issue.Code != "disconnected" {
				return false
			}
		}
		return true
	}
	ok := local.walk(0)
	if ok {
		s.budget.repairs++
	}
	if s.budget.used >= s.budget.limit || s.budget.ctx.Err() != nil {
		s.limited = true
	}
	return local.result, ok
}

func layoutComponents(placements []Placement, catalog PartCatalog) ([]int, int) {
	parents := make([]int, len(placements))
	ids := map[string]int{}
	for i, p := range placements {
		parents[i] = i
		ids[p.ID] = i
	}
	root := func(i int) int {
		for parents[i] != i {
			i = parents[i]
		}
		return i
	}
	count := len(placements)
	for _, connection := range deriveExactConnections(placements, catalog) {
		a, b := root(ids[connection.APlacementID]), root(ids[connection.BPlacementID])
		if a != b {
			parents[a] = b
			count--
		}
	}
	for i := range parents {
		parents[i] = root(i)
	}
	return parents, count
}
