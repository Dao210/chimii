package build

// platformPiers bounds the unsupported gap to four studs. A fixed six-stud
// pitch fits the reviewed eight-stud bricks with two-stud end bearings. Exact
// catalog connectors and earlier-step support are still checked by the compiler.
func platformPiers(length int) []int {
	positions := []int{0}
	for at := 6; at < length-2; at += 6 {
		positions = append(positions, at)
	}
	if positions[len(positions)-1] != length-2 {
		positions = append(positions, length-2)
	}
	return positions
}

func supportedPlatformCells(size DesignVector) map[DesignVector]bool {
	cells := map[DesignVector]bool{}
	capBottom := size.Y - 6
	zPiers := platformPiers(size.Z)
	for _, x := range platformPiers(size.X) {
		for _, z := range zPiers {
			for y := 0; y < capBottom; y++ {
				for dx := 0; dx < 2; dx++ {
					for dz := 0; dz < 2; dz++ {
						cells[DesignVector{x + dx, y, z + dz}] = true
					}
				}
			}
		}
	}
	for x := 0; x < size.X; x++ {
		for z := 0; z < size.Z; z++ {
			for y := capBottom; y < size.Y; y++ {
				// Build the lower cap as transverse beams over pier rows,
				// then the upper layer as a full deck. A solid lower layer
				// would demand bricks in empty gaps before the beams exist.
				keep := y >= capBottom+3
				for _, pier := range zPiers {
					keep = keep || z >= pier && z < pier+2
				}
				if keep {
					cells[DesignVector{x, y, z}] = true
				}
			}
		}
	}
	return cells
}
