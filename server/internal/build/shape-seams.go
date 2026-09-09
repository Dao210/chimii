package build

// seamDepth follows BrickGPT mesh2brick's cumulative-gap priority, expressed in
// Chimii's plate units. Crossing a persistent joint is preferable to extending
// it upward. This is a search heuristic, not a force or stability calculation.
// See THIRD_PARTY_NOTICES.md for the source and license.
func (s *shapeSearch) seamDepth(cell DesignVector, sx, sz int) int {
	total := 0
	for x := 0; x < sx; x++ {
		for z := 0; z < sz; z++ {
			a := DesignVector{cell.X + x, cell.Y, cell.Z + z}
			if x+1 < sx {
				total += s.jointDepth(a, DesignVector{a.X + 1, a.Y, a.Z})
			}
			if z+1 < sz {
				total += s.jointDepth(a, DesignVector{a.X, a.Y, a.Z + 1})
			}
		}
	}
	return total
}

func (s *shapeSearch) jointDepth(a, b DesignVector) int {
	depth := 0
	for a.Y, b.Y = a.Y-1, b.Y-1; a.Y >= 0; a.Y, b.Y = a.Y-1, b.Y-1 {
		left, leftOK := s.occupied[a]
		right, rightOK := s.occupied[b]
		// Empty space is a requested opening, not a brick joint to fill.
		if !leftOK || !rightOK || left == right {
			break
		}
		depth++
	}
	return depth
}

// criticalSeamRegion grows a bounded neighborhood around a disconnected seam.
// Face contact only chooses which parts may be retiled; acceptance still uses
// exact connectors. Unlike upstream's random k-ring selection, order is stable.
func criticalSeamRegion(placements []Placement, catalog PartCatalog, a, b int) []int {
	const maxRegionParts = 8
	region := []int{a, b}
	seen := map[int]bool{a: true, b: true}
	for _, seed := range []int{a, b} {
		for i, p := range placements {
			if seen[i] || !solidPartsTouch(placements[seed], p, catalog) {
				continue
			}
			seen[i] = true
			region = append(region, i)
			if len(region) == maxRegionParts {
				return region
			}
		}
	}
	return region
}

func solidPartsTouch(a, b Placement, catalog PartCatalog) bool {
	as, bs := orientedSizeWith(a, catalog), orientedSizeWith(b, catalog)
	ax, ay, az := a.X+as.x, topYWith(a, catalog), a.Z+as.z
	bx, by, bz := b.X+bs.x, topYWith(b, catalog), b.Z+bs.z
	x := a.X < bx && b.X < ax
	y := a.Y < by && b.Y < ay
	z := a.Z < bz && b.Z < az
	return ((ax == b.X || bx == a.X) && y && z) ||
		((ay == b.Y || by == a.Y) && x && z) ||
		((az == b.Z || bz == a.Z) && x && y)
}
