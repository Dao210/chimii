package ldrawsync

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const PartSemanticVersion = 1

type PartSemantics struct {
	PartKey                string
	Name                   string
	Category               string
	PopularityRank         int
	CertificationLevel     string
	AutoBuildEligible      bool
	GeometryProfile        string
	StudsX                 int
	StudsZ                 int
	PlatesY                int
	DefaultQuantity        int
	HasTopStuds            bool
	HasBottomReceptors     bool
	OriginYOffsetLDU       int
	OriginCenterZOffsetLDU int
}

type semanticConnector struct {
	Kind      string `json:"kind"`
	X         int    `json:"x_ldu"`
	Y         int    `json:"y_ldu"`
	Z         int    `json:"z_ldu"`
	Direction string `json:"direction"`
}

type legacySemantic struct {
	PartKey                string
	StudsX                 int
	StudsZ                 int
	PlatesY                int
	DefaultQuantity        int
	GeometryProfile        string
	OriginYOffsetLDU       int
	OriginCenterZOffsetLDU int
}

var legacySemantics = map[string]legacySemantic{
	"3001.dat":    {PartKey: "brick-2x4", StudsX: 4, StudsZ: 2, PlatesY: 3, DefaultQuantity: 18, GeometryProfile: "stud_tube_rect"},
	"3003.dat":    {PartKey: "brick-2x2", StudsX: 2, StudsZ: 2, PlatesY: 3, DefaultQuantity: 16, GeometryProfile: "stud_tube_rect"},
	"3004.dat":    {PartKey: "brick-1x2", StudsX: 2, StudsZ: 1, PlatesY: 3, DefaultQuantity: 20, GeometryProfile: "stud_tube_rect"},
	"3005.dat":    {PartKey: "brick-1x1", StudsX: 1, StudsZ: 1, PlatesY: 3, DefaultQuantity: 20, GeometryProfile: "stud_tube_rect"},
	"3020.dat":    {PartKey: "plate-2x4", StudsX: 4, StudsZ: 2, PlatesY: 1, DefaultQuantity: 12, GeometryProfile: "stud_tube_rect"},
	"3022.dat":    {PartKey: "plate-2x2", StudsX: 2, StudsZ: 2, PlatesY: 1, DefaultQuantity: 12, GeometryProfile: "stud_tube_rect"},
	"3023.dat":    {PartKey: "plate-1x2", StudsX: 2, StudsZ: 1, PlatesY: 1, DefaultQuantity: 16, GeometryProfile: "stud_tube_rect"},
	"3039.dat":    {PartKey: "slope-2x2", StudsX: 2, StudsZ: 2, PlatesY: 3, DefaultQuantity: 8, GeometryProfile: "legacy_special"},
	"4600.dat":    {PartKey: "wheel-holder-2x2", StudsX: 2, StudsZ: 2, PlatesY: 1, DefaultQuantity: 4, GeometryProfile: "legacy_special"},
	"4624c04.dat": {PartKey: "wheel", StudsX: 1, StudsZ: 1, PlatesY: 2, DefaultQuantity: 8, GeometryProfile: "legacy_special", OriginYOffsetLDU: 5, OriginCenterZOffsetLDU: 10},
}

var (
	simpleBrickPattern = regexp.MustCompile(`^Brick ([0-9]+) x ([0-9]+)(?: x ([0-9]+))?$`)
	simplePlatePattern = regexp.MustCompile(`^Plate ([0-9]+) x ([0-9]+)$`)
	simpleTilePattern  = regexp.MustCompile(`^Tile ([0-9]+) x ([0-9]+)(?: with Groove)?$`)
)

// DerivePartSemantics is intentionally conservative. Only exact rectangular
// brick, plate and tile names receive generated geometry/connection semantics.
// Everything else remains asset_only until a reviewed profile is added.
func DerivePartSemantics(part StarterKitPart) PartSemantics {
	id := normalizeName(part.LDrawID)
	semantics := PartSemantics{
		PartKey: "ldraw-" + strings.NewReplacer(".dat", "", "/", "-").Replace(id),
		Name:    strings.TrimSpace(part.Name), Category: strings.TrimSpace(part.Category),
		PopularityRank: part.Rank, CertificationLevel: "asset_only", GeometryProfile: "asset_only",
	}
	if legacy, ok := legacySemantics[id]; ok {
		semantics.PartKey = legacy.PartKey
		semantics.CertificationLevel = "certified"
		semantics.AutoBuildEligible = true
		semantics.GeometryProfile = legacy.GeometryProfile
		semantics.StudsX, semantics.StudsZ, semantics.PlatesY = legacy.StudsX, legacy.StudsZ, legacy.PlatesY
		semantics.DefaultQuantity = legacy.DefaultQuantity
		semantics.HasTopStuds = id != "4624c04.dat"
		semantics.HasBottomReceptors = id != "4624c04.dat"
		semantics.OriginYOffsetLDU = legacy.OriginYOffsetLDU
		semantics.OriginCenterZOffsetLDU = legacy.OriginCenterZOffsetLDU
		return semantics
	}

	if part.Category == "Bricks" {
		if x, z, height, ok := matchRect(simpleBrickPattern, part.Name); ok {
			semantics.CertificationLevel, semantics.GeometryProfile = "basic", "stud_tube_rect"
			semantics.AutoBuildEligible = true
			semantics.StudsX, semantics.StudsZ, semantics.PlatesY = x, z, height*3
			semantics.DefaultQuantity = 12
			semantics.HasTopStuds, semantics.HasBottomReceptors = true, true
		}
		return semantics
	}
	if part.Category == "Plates" {
		if x, z, _, ok := matchRect(simplePlatePattern, part.Name); ok {
			semantics.CertificationLevel, semantics.GeometryProfile = "basic", "stud_tube_rect"
			semantics.AutoBuildEligible = true
			semantics.StudsX, semantics.StudsZ, semantics.PlatesY = x, z, 1
			semantics.DefaultQuantity = 12
			semantics.HasTopStuds, semantics.HasBottomReceptors = true, true
		}
		return semantics
	}
	if part.Category == "Tiles" {
		if x, z, _, ok := matchRect(simpleTilePattern, part.Name); ok {
			semantics.CertificationLevel, semantics.GeometryProfile = "basic", "tile_rect"
			semantics.StudsX, semantics.StudsZ, semantics.PlatesY = x, z, 1
			semantics.DefaultQuantity = 8
			semantics.HasBottomReceptors = true
		}
	}
	return semantics
}

func matchRect(pattern *regexp.Regexp, name string) (x, z, height int, ok bool) {
	match := pattern.FindStringSubmatch(strings.TrimSpace(name))
	if len(match) < 3 {
		return 0, 0, 0, false
	}
	x, _ = strconv.Atoi(match[1])
	z, _ = strconv.Atoi(match[2])
	height = 1
	if len(match) > 3 && match[3] != "" {
		height, _ = strconv.Atoi(match[3])
	}
	return x, z, height, x > 0 && z > 0 && height > 0
}

func (semantics PartSemantics) ConnectionsJSON() []byte {
	connectors := make([]semanticConnector, 0, semantics.StudsX*semantics.StudsZ*2)
	for x := 0; x < semantics.StudsX; x++ {
		for z := 0; z < semantics.StudsZ; z++ {
			xLDU := x*20 - (semantics.StudsX-1)*10
			zLDU := z*20 - (semantics.StudsZ-1)*10
			if semantics.HasTopStuds {
				connectors = append(connectors, semanticConnector{Kind: "stud", X: xLDU, Y: 0, Z: zLDU, Direction: "up"})
			}
			if semantics.HasBottomReceptors {
				connectors = append(connectors, semanticConnector{Kind: "receptor", X: xLDU, Y: semantics.PlatesY * 8, Z: zLDU, Direction: "down"})
			}
		}
	}
	raw, _ := json.Marshal(connectors)
	return raw
}

func (semantics PartSemantics) OccupancyJSON() []byte {
	raw, _ := json.Marshal(map[string]any{
		"profile":  semantics.GeometryProfile,
		"studs_x":  semantics.StudsX,
		"studs_z":  semantics.StudsZ,
		"plates_y": semantics.PlatesY,
	})
	return raw
}

func BoundsJSON(bounds [6]float64) []byte {
	raw, _ := json.Marshal(map[string]float64{
		"min_x": bounds[0], "min_y": bounds[1], "min_z": bounds[2],
		"max_x": bounds[3], "max_y": bounds[4], "max_z": bounds[5],
	})
	return raw
}

func (semantics PartSemantics) Validate() error {
	if semantics.PartKey == "" || semantics.PopularityRank <= 0 {
		return fmt.Errorf("invalid semantic identity for %q", semantics.Name)
	}
	if semantics.CertificationLevel != "asset_only" && (semantics.StudsX <= 0 || semantics.StudsZ <= 0 || semantics.PlatesY <= 0) {
		return fmt.Errorf("certified part %s has invalid dimensions", semantics.PartKey)
	}
	return nil
}
