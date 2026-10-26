package build

import "fmt"

// ConstructionCapabilities exposes construction techniques, not a subject whitelist.
func ConstructionCapabilities() []map[string]string {
	return []map[string]string{
		{"kind": "solid", "usage": "rectangular slab, beam, floor, shelf, hull or limb; dimensions in studs/plates"},
		{"kind": "platform", "usage": "supported raised deck: 2x2 pillars at both edges and at most 6-stud pitch, with a 6-plate cap. X/Z>=4, Y>=9 and divisible by 3; Y includes the pillars and cap. Put on a full foundation of the same footprint; attach payloads to its top. Suitable for open pavilions, display stands and wing supports. Not a room or stairs; reserve the pillar grid's space."},
		{"kind": "paint", "usage": "real colored-brick patch INSIDE its parent, typically height 3/6 plates and depth 1 for eyes/panels. Optional openings on a thin patch preserve a window/door hole while coloring its surrounding frame: same {x,y,width,height} as parent face, no face field, depth<=2. An 8x12x2 face with window {x:2,y:3,width:4,height:6} can take front-attached paint of that same size/opening. Does not add/fill volume; never paint a solid rectangle over a window."},
		{"kind": "wall", "usage": "wall with openings along its longer horizontal X/Z axis (openings.x is distance along that axis); opening.x>=2; opening.x+width<=wall.length-2; opening.y+height<=wall.height-3; valid 8x9x2 wall: opening {x:2,y:0,width:4,height:6}; widths<=max_span_studs"},
		{"kind": "frame", "usage": "hollow rectangular perimeter, 2 studs thick, size.X AND size.Z >=6; optional openings with face front/back/left/right, x distance along that wall, y plate height; real windows require openings, never paint; floors/roofs are separate solids; compact 10x12x10 rooms have a 6-stud interior gap"},
		{"kind": "storey", "usage": "complete building level: frame walls, interior 2x2 pier grid at 6-stud pitch, transverse beams and top deck. Size.Y includes 6-plate ceiling; X/Z>=6, Y>=12 divisible by 3. Face openings must end <=Y-9 (wall lintel). Optional ceiling_openings:[{x,z,width,depth}] cut BOTH ceiling layers for actual stair access, inside 2-stud exterior rim. Do not add another overlapping floor. Example 14x18x18 level on a full base: stairs 2x18x12 also on that base, offset x=-4,z=0, aligned ceiling opening {x:2,z:3,width:2,depth:12}; next storey attaches to this level top."},
		{"kind": "stairs", "usage": "solid staircase along +Z; 2-stud treads, 3-plate risers; size.y=3*(size.z/2)"},
		{"kind": "taper", "usage": "stepped mass; each 3 plates inset 1 stud on all X/Z sides; require min(size.x,size.z)>=2+2*(size.y/3-1); valid ear size 4x6x4 or 6x9x6 (x,y,z), invalid 2x6x2"},
		{"kind": "roof", "usage": "stepped roof; each 3 plates inset 1 stud on both X sides; require size.x>=2+2*(size.y/3-1); valid size 12x12x10 (x,y,z)"},
		{"kind": "ellipse", "usage": "elliptical horizontal footprint extruded in Y; combine tiers for heads and bodies"},
		{"kind": "shape", "usage": "local DesignSpec v1 with box/ellipse/polygon additions and subtraction; preserve holes and feature colors"},
		{"kind": "wheelbase", "usage": "rolling chassis; even width/depth 4..16 studs, height exactly 5 plates, local Y=0; body attaches at Y=5 over central width (1..width-1), full depth; tires installed after body"},
	}
}

// constructionCells compiles parameters into target volume. No model-supplied
// part coordinates or executable generator code is evaluated.
func constructionCells(m Construction) (map[DesignVector]bool, []Placement, error) {
	cells := map[DesignVector]bool{}
	if m.Kind == "paint" && !sceneID(m.Parent) || m.Kind != "paint" && m.Parent != "" {
		return nil, nil, designError("only paint has a required parent construction")
	}
	if len(m.CeilingOpenings) > 0 && m.Kind != "storey" {
		return nil, nil, designError("ceiling openings are supported only by storey constructions")
	}
	if m.Kind == "wheelbase" {
		s := m.Size
		if m.Design != nil || len(m.Openings) > 0 || s.X < 4 || s.X > 16 || s.Z < 4 || s.Z > 16 || s.X%2 != 0 || s.Z%2 != 0 || s.Y != 5 || m.Position.Y != 0 {
			return nil, nil, designError("wheelbase requires even width/depth 4..16, height 5, at ground level")
		}
		parts := []Placement{}
		for x := 1; x < s.X-1; x += 2 {
			for z := 0; z < s.Z; z += 2 {
				id := "plate-2x2"
				if (x == 1 || x == s.X-3) && (z == 0 || z == s.Z-2) {
					id = "wheel-holder-2x2"
				}
				parts = append(parts, Placement{ID: fmt.Sprintf("under-%d-%d", x, z), PartID: id, X: x, Z: z, Step: 1})
			}
		}
		for _, z := range []int{0, s.Z - 2} {
			parts = append(parts, Placement{ID: fmt.Sprintf("wheel-left-%d", z), PartID: "wheel", X: 0, Z: z, Rotation: 90, Step: 3}, Placement{ID: fmt.Sprintf("wheel-right-%d", z), PartID: "wheel", X: s.X - 1, Z: z, Rotation: 270, Step: 3})
		}
		for x := 1; x < s.X-1; x++ {
			for z := 0; z < s.Z; z++ {
				for y := 1; y < 5; y++ {
					cells[DesignVector{x, y, z}] = true
				}
			}
		}
		return cells, parts, nil
	}
	if m.Kind == "shape" {
		if m.Design == nil || len(m.Openings) > 0 {
			return nil, nil, designError("shape requires a local design")
		}
		target, err := RasterizeDesign(*m.Design)
		if err != nil {
			return nil, nil, err
		}
		for p := range target {
			cells[p] = true
		}
		return cells, nil, nil
	}
	if m.Design != nil || len(m.Openings) > 0 && m.Kind != "wall" && m.Kind != "frame" && m.Kind != "paint" && m.Kind != "storey" {
		return nil, nil, designError("unexpected construction parameters")
	}
	s := m.Size
	if s.X < 1 || s.X > 32 || s.Z < 1 || s.Z > 32 || s.Y < 1 || s.Y > 96 || s.X*s.Y*s.Z > 8192 {
		return nil, nil, designError("construction size outside supported bounds")
	}
	switch m.Kind {
	case "solid", "ellipse", "paint":
		if m.Kind == "paint" && len(m.Openings) > 0 && min(s.X, s.Z) > 2 {
			return nil, nil, designError("paint with openings must be a thin wall-face patch, at most 2 studs deep")
		}
	case "platform":
		if s.X < 4 || s.Z < 4 || s.Y < 9 || s.Y%3 != 0 {
			return nil, nil, designError("platform requires X/Z>=4 and Y>=9 divisible by 3, including a 6-plate cap")
		}
		return supportedPlatformCells(s), nil, nil
	case "storey":
		return supportedStoreyCells(m)
	case "wall":
		if min(s.X, s.Z) > 2 {
			return nil, nil, designError("wall must be at most 2 studs thick; use frame with face openings for a hollow room/cabin")
		}
	case "frame":
		if s.X < 6 || s.Z < 6 {
			return nil, nil, designError("frame needs an interior opening")
		}
	case "stairs":
		if s.Z%2 != 0 || s.Y != 3*(s.Z/2) {
			return nil, nil, designError("stairs require 2-stud treads and 3-plate risers")
		}
	case "taper":
		if s.Y%3 != 0 || s.X-2*(s.Y/3-1) < 2 || s.Z-2*(s.Y/3-1) < 2 {
			return nil, nil, designError(fmt.Sprintf("taper size (%d,%d,%d) invalid: Y must be divisible by 3 and both X,Z must be >=%d for this Y. For a 6-plate ear use 4x6x4; for a 9-plate ear use 6x9x6", s.X, s.Y, s.Z, 2+2*(s.Y/3-1)))
		}
	case "roof":
		if s.Y%3 != 0 || s.X-2*(s.Y/3-1) < 2 {
			return nil, nil, designError(fmt.Sprintf("roof size (%d,%d,%d) invalid: Y must be divisible by 3 and X must be >=%d for this Y", s.X, s.Y, s.Z, 2+2*(s.Y/3-1)))
		}
	default:
		return nil, nil, designError("unknown construction kind")
	}
	if err := validateConstructionOpenings(m); err != nil {
		return nil, nil, err
	}
	for y := 0; y < s.Y; y++ {
		for x := 0; x < s.X; x++ {
			for z := 0; z < s.Z; z++ {
				keep := true
				switch m.Kind {
				case "ellipse":
					keep = shapeContains(ShapeNode{Kind: "ellipse", Size: s}, float64(x)+0.5, float64(z)+0.5)
				case "frame":
					keep = x < 2 || x >= s.X-2 || z < 2 || z >= s.Z-2
					for _, o := range m.Openings {
						along, onWall := x, z < 2
						switch o.Face {
						case "back":
							onWall = z >= s.Z-2
						case "left":
							along, onWall = z, x < 2
						case "right":
							along, onWall = z, x >= s.X-2
						}
						if onWall && along >= o.X && along < o.X+o.Width && y >= o.Y && y < o.Y+o.Height {
							keep = false
						}
					}
				case "stairs":
					keep = y < 3*(z/2+1)
				case "taper":
					inset := y / 3
					keep = x >= inset && x < s.X-inset && z >= inset && z < s.Z-inset
				case "roof":
					inset := y / 3
					keep = x >= inset && x < s.X-inset
				case "wall", "paint":
					along := x
					if s.Z > s.X {
						along = z
					}
					for _, o := range m.Openings {
						if along >= o.X && along < o.X+o.Width && y >= o.Y && y < o.Y+o.Height {
							keep = false
						}
					}
				}
				if keep {
					cells[DesignVector{x, y, z}] = true
				}
			}
		}
	}
	return cells, nil, nil
}

func constructionOpeningAlongZ(m Construction, o WallOpening) bool {
	return (m.Kind == "wall" || m.Kind == "paint") && m.Size.Z > m.Size.X || (m.Kind == "frame" || m.Kind == "storey") && (o.Face == "left" || o.Face == "right")
}

func validateConstructionOpenings(m Construction) error {
	if len(m.Openings) > 24 {
		return designError("too many construction openings")
	}
	for i, o := range m.Openings {
		if (m.Kind == "wall" || m.Kind == "paint") && o.Face != "" || m.Kind == "frame" && o.Face != "" && o.Face != "front" && o.Face != "back" && o.Face != "left" && o.Face != "right" {
			return designError("opening face must be front/back/left/right for a frame, or omitted for a wall")
		}
		length := m.Size.X
		if constructionOpeningAlongZ(m, o) {
			length = m.Size.Z
		}
		if o.X < 2 || o.Y < 0 || o.Width < 1 || o.Width > 6 || o.Height < 1 || o.X+o.Width > length-2 || o.Y+o.Height > m.Size.Y-3 {
			return designError(fmt.Sprintf("%s opening %d (%s,%d,%d,%d,%d) needs 2-stud end piers, width <=6 and a 3-plate lintel; length=%d height=%d", m.ID, i, o.Face, o.X, o.Y, o.Width, o.Height, length, m.Size.Y))
		}
		for _, previous := range m.Openings[:i] {
			face := func(s string) string {
				if s == "" {
					return "front"
				}
				return s
			}
			if face(previous.Face) == face(o.Face) && previous.Y < o.Y+o.Height && o.Y < previous.Y+previous.Height && previous.X < o.X+o.Width+2 && o.X < previous.X+previous.Width+2 {
				return designError("openings on the same wall need a 2-stud pier between them: " + m.ID)
			}
		}
	}
	return nil
}
