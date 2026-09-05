package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
)

const DesignVersion = 1
const ShapeGeneratorVersion = "shape-layout-v1"
const BuildErrorSearchLimit = "BUILD_SEARCH_LIMIT"
const MaxDesignShapes = 48
const maxDesignCells = 8192

// DesignSpec describes target volumes, never physical part coordinates. Each
// target is sampled at one stud by one plate; real catalog parts tile the result.
type DesignSpec struct {
	Version int         `json:"version"`
	Mode    string      `json:"mode"`
	Shapes  []ShapeNode `json:"shapes"`
}

type DesignVector struct {
	X int `json:"x"`
	Y int `json:"y"`
	Z int `json:"z"`
}
type DesignPoint struct {
	X int `json:"x"`
	Z int `json:"z"`
}
type ShapeRepeat struct {
	Count  int          `json:"count"`
	Offset DesignVector `json:"offset"`
}
type ShapeNode struct {
	ID        string        `json:"id"`
	Label     string        `json:"label"`
	Kind      string        `json:"kind"`
	Operation string        `json:"operation"`
	Position  DesignVector  `json:"position"`
	Size      DesignVector  `json:"size"`
	Color     int           `json:"color"`
	Points    []DesignPoint `json:"points,omitempty"`
	Repeat    *ShapeRepeat  `json:"repeat,omitempty"`
}

// BuildDocument is a validated revision, shared by rendering, edits and export.
// Editing creates a new creation; an in-progress build never changes underneath it.
type BuildDocument struct {
	Version          int          `json:"version"`
	Design           DesignSpec   `json:"design"`
	DesignHash       string       `json:"design_hash"`
	ParentCreationID string       `json:"parent_creation_id,omitempty"`
	ParentHash       string       `json:"parent_hash,omitempty"`
	Solver           SolverReport `json:"solver"`
}
type SolverReport struct {
	Status       string `json:"status"`
	Visited      int    `json:"visited"`
	TargetCells  int    `json:"target_cells"`
	MatchedCells int    `json:"matched_cells"`
}

type targetCell struct {
	Node  string
	Color int
}
type designTarget map[DesignVector]targetCell

func designError(message string) error {
	return &BuildError{Code: BuildErrorRequirements, Cause: fmt.Errorf("%s", message)}
}

func DesignHash(design DesignSpec) string {
	raw, _ := json.Marshal(design)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// RasterizeDesign applies ordered union/subtraction. All constructive nodes must
// retain visible volume; a later operation cannot silently delete a requested feature.
func RasterizeDesign(design DesignSpec) (designTarget, error) {
	if design.Version != DesignVersion || design.Mode != "static" || len(design.Shapes) == 0 || len(design.Shapes) > MaxDesignShapes {
		return nil, designError("invalid design version, mode or shape count")
	}
	target := designTarget{}
	seen := map[string]bool{}
	work := 0
	for _, node := range design.Shapes {
		if node.ID == "" || len(node.ID) > 48 || seen[node.ID] || len([]rune(node.Label)) > 80 {
			return nil, designError("invalid or duplicate shape id")
		}
		seen[node.ID] = true
		if node.Position.X < -16 || node.Position.X > 16 || node.Position.Z < -16 || node.Position.Z > 16 || node.Position.Y < 0 || node.Position.Y > 47 {
			return nil, designError("invalid shape position")
		}
		if node.Operation != "add" && node.Operation != "subtract" {
			return nil, designError("unknown shape operation")
		}
		if !IsAllowedColor(node.Color) || node.Size.X < 1 || node.Size.X > 32 || node.Size.Z < 1 || node.Size.Z > 32 || node.Size.Y < 1 || node.Size.Y > 48 {
			return nil, designError("invalid shape size or color")
		}
		switch node.Kind {
		case "box", "ellipse":
			if len(node.Points) != 0 {
				return nil, designError("points require a polygon")
			}
		case "polygon":
			if len(node.Points) < 3 || len(node.Points) > 32 {
				return nil, designError("polygon needs 3 to 32 vertices")
			}
			for _, p := range node.Points {
				if p.X < 0 || p.Z < 0 || p.X > node.Size.X || p.Z > node.Size.Z {
					return nil, designError("polygon vertex outside shape bounds")
				}
			}
		default:
			return nil, designError("unknown shape primitive")
		}
		count, offset := 1, (DesignVector{})
		if node.Repeat != nil {
			count, offset = node.Repeat.Count, node.Repeat.Offset
			if count < 1 || count > 24 || offset.X < -32 || offset.X > 32 || offset.Z < -32 || offset.Z > 32 || offset.Y < -48 || offset.Y > 48 {
				return nil, designError("invalid shape repetition")
			}
		}
		for n := 0; n < count; n++ {
			origin := DesignVector{node.Position.X + n*offset.X, node.Position.Y + n*offset.Y, node.Position.Z + n*offset.Z}
			if origin.X < -16 || origin.Z < -16 || origin.Y < 0 || origin.X+node.Size.X > 17 || origin.Z+node.Size.Z > 17 || origin.Y+node.Size.Y > 48 {
				return nil, designError("shape outside supported design bounds")
			}
			work += node.Size.X * node.Size.Y * node.Size.Z
			if work > maxDesignCells*16 {
				return nil, designError("design raster budget exceeded")
			}
			for x := 0; x < node.Size.X; x++ {
				for z := 0; z < node.Size.Z; z++ {
					if !shapeContains(node, float64(x)+0.5, float64(z)+0.5) {
						continue
					}
					for y := 0; y < node.Size.Y; y++ {
						p := DesignVector{origin.X + x, origin.Y + y, origin.Z + z}
						if node.Operation == "subtract" {
							delete(target, p)
						} else {
							target[p] = targetCell{node.ID, node.Color}
						}
					}
				}
			}
		}
	}
	if len(target) == 0 || len(target) > maxDesignCells {
		return nil, designError("empty or oversized target volume")
	}
	visible := map[string]bool{}
	for _, cell := range target {
		visible[cell.Node] = true
	}
	for _, node := range design.Shapes {
		if node.Operation == "add" && !visible[node.ID] {
			return nil, designError("shape was completely removed: " + node.ID)
		}
	}
	return target, nil
}

func shapeContains(node ShapeNode, x, z float64) bool {
	switch node.Kind {
	case "box":
		return true
	case "ellipse":
		dx, dz := (x-float64(node.Size.X)/2)/(float64(node.Size.X)/2), (z-float64(node.Size.Z)/2)/(float64(node.Size.Z)/2)
		return dx*dx+dz*dz <= 1+1e-9
	case "polygon":
		inside := false
		for i, a := range node.Points {
			b := node.Points[(i+1)%len(node.Points)]
			ax, az, bx, bz := float64(a.X), float64(a.Z), float64(b.X), float64(b.Z)
			cross := (x-ax)*(bz-az) - (z-az)*(bx-ax)
			if math.Abs(cross) < 1e-9 && x >= math.Min(ax, bx) && x <= math.Max(ax, bx) && z >= math.Min(az, bz) && z <= math.Max(az, bz) {
				return true
			}
			if (az > z) != (bz > z) && x < (bx-ax)*(z-az)/(bz-az)+ax {
				inside = !inside
			}
		}
		return inside
	}
	return false
}

func sortedTargetCells(target designTarget) []DesignVector {
	cells := make([]DesignVector, 0, len(target))
	for p := range target {
		cells = append(cells, p)
	}
	sort.Slice(cells, func(i, j int) bool {
		a, b := cells[i], cells[j]
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		if a.Z != b.Z {
			return a.Z < b.Z
		}
		return a.X < b.X
	})
	return cells
}
