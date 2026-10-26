package build

func supportedStoreyCells(module Construction) (map[DesignVector]bool, []Placement, error) {
	size := module.Size
	if size.X < 6 || size.Z < 6 || size.Y < 12 || size.Y%3 != 0 || len(module.CeilingOpenings) > 8 {
		return nil, nil, designError("storey requires X/Z>=6 and Y>=12 divisible by 3 including its 6-plate ceiling, at most 8 stair/access openings")
	}
	for i, opening := range module.CeilingOpenings {
		if opening.X < 2 || opening.Z < 2 || opening.Width < 1 || opening.Depth < 1 || opening.X+opening.Width > size.X-2 || opening.Z+opening.Depth > size.Z-2 {
			return nil, nil, designError("ceiling opening must stay inside a 2-stud exterior rim: " + module.ID)
		}
		for _, previous := range module.CeilingOpenings[:i] {
			if previous.X < opening.X+opening.Width && opening.X < previous.X+previous.Width && previous.Z < opening.Z+opening.Depth && opening.Z < previous.Z+previous.Depth {
				return nil, nil, designError("ceiling openings must not overlap: " + module.ID)
			}
		}
	}
	walls := module
	walls.Kind, walls.Size.Y, walls.CeilingOpenings = "frame", size.Y-6, nil
	cells, _, err := constructionCells(walls)
	if err != nil {
		return nil, nil, err
	}
	for p := range supportedPlatformCells(size) {
		if p.Y < size.Y-6 {
			// Exterior piers already belong to the perimeter walls and must
			// never fill their doors/windows. Only add interior supports.
			if p.X < 2 || p.X >= size.X-2 || p.Z < 2 || p.Z >= size.Z-2 {
				continue
			}
		} else {
			cut := false
			for _, opening := range module.CeilingOpenings {
				cut = cut || p.X >= opening.X && p.X < opening.X+opening.Width && p.Z >= opening.Z && p.Z < opening.Z+opening.Depth
			}
			if cut {
				continue
			}
		}
		cells[p] = true
	}
	// Add transverse trimmers before and after each access hole. The upper
	// deck's short edge must bear on a beam; it cannot span across a hole that
	// was deliberately kept empty. Trimmers never cover any access opening.
	for _, opening := range module.CeilingOpenings {
		for z := 0; z < size.Z; z++ {
			if !(z >= opening.Z-2 && z < opening.Z || z >= opening.Z+opening.Depth && z < opening.Z+opening.Depth+2) {
				continue
			}
			for x := 0; x < size.X; x++ {
				cut := false
				for _, hole := range module.CeilingOpenings {
					cut = cut || x >= hole.X && x < hole.X+hole.Width && z >= hole.Z && z < hole.Z+hole.Depth
				}
				if !cut {
					for y := size.Y - 6; y < size.Y-3; y++ {
						cells[DesignVector{x, y, z}] = true
					}
				}
			}
		}
	}
	return cells, nil, nil
}
