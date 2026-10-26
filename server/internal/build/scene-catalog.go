package build

// SceneCatalogProfile describes certified geometry capabilities, never owned
// inventory, retail availability or a strength guarantee.
type SceneCatalogProfile struct {
	AutomaticParts            int  `json:"automatic_parts"`
	RectangularParts          int  `json:"rectangular_parts"`
	MaxRectangularLengthStuds int  `json:"max_rectangular_length_studs"`
	FixedPierClearGapStuds    int  `json:"fixed_pier_clear_gap_studs"`
	FullPierGridBeams         bool `json:"full_pier_grid_beams"`
}

func DescribeSceneCatalog(catalog PartCatalog) SceneCatalogProfile {
	p := SceneCatalogProfile{FixedPierClearGapStuds: 4}
	for _, part := range catalog {
		if partIsMechanicallyCertified(part) {
			p.AutomaticParts++
		}
		if !IsShapePartEligible(part) {
			continue
		}
		p.RectangularParts++
		p.MaxRectangularLengthStuds = max(p.MaxRectangularLengthStuds, part.StudsX, part.StudsZ)
	}
	// A four-stud void needs at least one stud of bearing at each end.
	// This is only a necessary catalog condition; final layout is still checked.
	p.FullPierGridBeams = p.MaxRectangularLengthStuds >= p.FixedPierClearGapStuds+2
	return p
}
