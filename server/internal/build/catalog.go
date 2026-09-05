package build

import "fmt"

const (
	StarterKitID           = "chimii-starter-v1"
	CatalogRelease         = "2026-08"
	CatalogArchiveSHA256   = "d2a695868ed2b3957c45b022a6451908edab22cc043179dd61d18dd382b35e11"
	CatalogSourceURL       = "https://library.ldraw.org/library/updates/complete.zip"
	ModuleLibraryVersion   = "chimii-construction-modules-v2"
	CompilerVersion        = "build-compiler-v3"
	ValidatorVersion       = "build-validator-v3"
	GeneratorVersion       = "certified-module-grammar-v1"
	PhysicsProfileVersion  = "conservative-static-v1"
	ConnectorSchemaVersion = 2
)

var CatalogVersion = ComposeLDrawCatalogVersion(CatalogRelease, CatalogArchiveSHA256)

func ComposeLDrawCatalogVersion(release, archiveSHA256 string) string {
	if archiveSHA256 == "" {
		return "ldraw-official-" + release + "-unknown"
	}
	return fmt.Sprintf("ldraw-official-%s-%s", release, archiveSHA256[:min(12, len(archiveSHA256))])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// StarterCatalog is deliberately small and versioned. The application never
// asks a model to invent a part number: every placement must resolve here.
var StarterCatalog = PartCatalog{
	"brick-2x4":        certifiedRectPart(PartSpec{ID: "brick-2x4", Name: "Brick 2 x 4", Category: "brick", LDrawID: "3001.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 4, StudsZ: 2, PlatesY: 3, Quantity: 18}),
	"brick-2x2":        certifiedRectPart(PartSpec{ID: "brick-2x2", Name: "Brick 2 x 2", Category: "brick", LDrawID: "3003.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 3, Quantity: 16}),
	"brick-1x2":        certifiedRectPart(PartSpec{ID: "brick-1x2", Name: "Brick 1 x 2", Category: "brick", LDrawID: "3004.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 1, PlatesY: 3, Quantity: 20}),
	"brick-1x1":        certifiedRectPart(PartSpec{ID: "brick-1x1", Name: "Brick 1 x 1", Category: "brick", LDrawID: "3005.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 1, StudsZ: 1, PlatesY: 3, Quantity: 20}),
	"plate-2x4":        certifiedRectPart(PartSpec{ID: "plate-2x4", Name: "Plate 2 x 4", Category: "plate", LDrawID: "3020.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 4, StudsZ: 2, PlatesY: 1, Quantity: 12}),
	"plate-2x2":        certifiedRectPart(PartSpec{ID: "plate-2x2", Name: "Plate 2 x 2", Category: "plate", LDrawID: "3022.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 1, Quantity: 12}),
	"plate-1x2":        certifiedRectPart(PartSpec{ID: "plate-1x2", Name: "Plate 1 x 2", Category: "plate", LDrawID: "3023.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 1, PlatesY: 1, Quantity: 16}),
	"slope-2x2":        certifiedSlopePart(PartSpec{ID: "slope-2x2", Name: "Roof Slope 2 x 2", Category: "slope", LDrawID: "3039.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 3, Quantity: 8}),
	"wheel-holder-2x2": certifiedWheelHolderPart(PartSpec{ID: "wheel-holder-2x2", Name: "Plate 2 x 2 with 2 Wheel Pins", Category: "wheel", LDrawID: "4600.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 1, Quantity: 4}),
	// The shortcut contains a compatible rim and tyre. Its axle origin is
	// calibrated 3 LDU below the holder top; the Z offset aligns the round wheel with the
	// holder's stud-row center while occupancy stays on the integer stud grid.
	"wheel": certifiedWheelPart(PartSpec{ID: "wheel", Name: "Wheel Rim 6.4 x 8 with Tyre 8/75 x 8", Category: "wheel", LDrawID: "4624c04.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 1, StudsZ: 1, PlatesY: 2, Quantity: 8, OriginYOffsetLDU: 13, OriginCenterZOffsetLDU: 10}),
}

func certifiedRectPart(part PartSpec) PartSpec {
	part.CertificationLevel = "certified"
	part.AutoBuildEligible = true
	part.GeometryProfile = "stud_tube_rect"
	part.HasTopStuds = true
	part.HasBottomReceptors = true
	part.Bounds = PartBounds{MinX: float64(-part.StudsX * 10), MinY: 0, MinZ: float64(-part.StudsZ * 10), MaxX: float64(part.StudsX * 10), MaxY: float64(part.PlatesY * 8), MaxZ: float64(part.StudsZ * 10)}
	part.Occupancy = PartOccupancy{Profile: part.GeometryProfile, GroundContactProfile: "footprint", StudsX: part.StudsX, StudsZ: part.StudsZ, PlatesY: part.PlatesY}
	part.Connectors = rectangularConnectors(part.StudsX, part.StudsZ, part.PlatesY, true, true)
	return part
}

func certifiedSlopePart(part PartSpec) PartSpec {
	part.CertificationLevel = "certified"
	part.AutoBuildEligible = true
	part.GeometryProfile = "legacy_special"
	part.HasBottomReceptors = true
	part.Bounds = PartBounds{MinX: float64(-part.StudsX * 10), MinY: 0, MinZ: float64(-part.StudsZ * 10), MaxX: float64(part.StudsX * 10), MaxY: float64(part.PlatesY * 8), MaxZ: float64(part.StudsZ * 10)}
	part.Occupancy = PartOccupancy{Profile: part.GeometryProfile, GroundContactProfile: "footprint", StudsX: part.StudsX, StudsZ: part.StudsZ, PlatesY: part.PlatesY}
	// The current templates use this slope only as a terminal detail. Its full
	// bottom receptor grid is certified; top studs remain unavailable until a
	// reviewed orientation-aware profile is added.
	part.Connectors = rectangularConnectors(part.StudsX, part.StudsZ, part.PlatesY, false, true)
	return part
}

func certifiedWheelHolderPart(part PartSpec) PartSpec {
	part = certifiedRectPart(part)
	part.GeometryProfile = "legacy_special"
	part.Occupancy.Profile = part.GeometryProfile
	part.Connectors = append(part.Connectors,
		PartConnector{ID: "wheel-pin-west", Kind: "wheel_pin", X: -30, Y: 5, Direction: "west", CapacityUnits: 2},
		PartConnector{ID: "wheel-pin-east", Kind: "wheel_pin", X: 30, Y: 5, Direction: "east", CapacityUnits: 2},
	)
	return part
}

func certifiedWheelPart(part PartSpec) PartSpec {
	part.CertificationLevel = "certified"
	part.AutoBuildEligible = true
	part.GeometryProfile = "legacy_special"
	part.Bounds = PartBounds{MinX: -10, MinY: -10, MinZ: -10, MaxX: 10, MaxY: 10, MaxZ: 10}
	part.Occupancy = PartOccupancy{Profile: part.GeometryProfile, GroundContactProfile: "wheel_point", StudsX: part.StudsX, StudsZ: part.StudsZ, PlatesY: part.PlatesY}
	part.Connectors = []PartConnector{{ID: "wheel-hole", Kind: "wheel_hole", Direction: "north", CapacityUnits: 2}}
	return part
}

func rectangularConnectors(studsX, studsZ, platesY int, top, bottom bool) []PartConnector {
	connectors := make([]PartConnector, 0, studsX*studsZ*2)
	for x := 0; x < studsX; x++ {
		for z := 0; z < studsZ; z++ {
			xLDU := x*20 - (studsX-1)*10
			zLDU := z*20 - (studsZ-1)*10
			if top {
				connectors = append(connectors, PartConnector{ID: fmt.Sprintf("stud-%d-%d", x, z), Kind: "stud", X: xLDU, Z: zLDU, Direction: "up", CapacityUnits: 1})
			}
			if bottom {
				connectors = append(connectors, PartConnector{ID: fmt.Sprintf("receptor-%d-%d", x, z), Kind: "receptor", X: xLDU, Y: platesY * 8, Z: zLDU, Direction: "down", CapacityUnits: 1})
			}
		}
	}
	return connectors
}

func CatalogCopy(parts PartCatalog) PartCatalog {
	copy := make(PartCatalog, len(parts))
	for key, part := range parts {
		part.Connectors = append([]PartConnector(nil), part.Connectors...)
		copy[key] = part
	}
	return copy
}

var allowedColors = map[int]bool{
	1:  true, // blue
	2:  true, // green
	4:  true, // red
	14: true, // yellow
	15: true, // white
	71: true, // light bluish gray
}

func IsAllowedColor(color int) bool { return allowedColors[color] }

func AllowedColorCodes() []int {
	return []int{1, 2, 4, 14, 15, 71}
}
