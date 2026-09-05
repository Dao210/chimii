package build

// repairSeams retiles neighboring parts on a disconnected seam. The target
// volume and every feature stay identical; only catalog part boundaries move.
func (s *shapeSearch) repairSeams(input []Placement) ([]Placement, bool) {
	placements := append([]Placement{}, input...)
	for round := 0; round < 48 && s.ctx.Err() == nil; round++ {
		components, count := layoutComponents(placements, s.catalog)
		if count == 1 {
			return placements, ValidateWithCatalog(placements, s.inventory, s.catalog).Buildable
		}
		improved := false
		for i := 0; i < len(placements) && !improved && s.ctx.Err() == nil; i++ {
			for j := i + 1; j < len(placements) && !improved && s.ctx.Err() == nil; j++ {
				if s.visited >= s.budget {
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
				local := &shapeSearch{ctx: s.ctx, target: s.target, cells: s.cells, parts: s.parts, catalog: s.catalog, inventory: s.inventory,
					remaining: s.inventory.quantities(), occupied: map[DesignVector]int{}, exactColors: s.exactColors, exactCount: s.exactCount, limit: min(160, s.budget-s.visited), variant: round % 3}
				for k, p := range placements {
					if k == i || k == j {
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
				if local.walk(0) {
					placements = local.result
					improved = true
				}
				s.visited += local.visited
			}
		}
		if !improved {
			return nil, false
		}
	}
	return nil, false
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
