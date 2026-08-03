package build

const (
	StarterKitID         = "chimii-starter-v1"
	CatalogVersion       = "ldraw-official-2026-05-29-6009f2e94204"
	ModuleLibraryVersion = "chimii-construction-modules-v1"
	CompilerVersion      = "build-compiler-v1"
	ValidatorVersion     = "build-validator-v1"
)

// StarterCatalog is deliberately small and versioned. The application never
// asks a model to invent a part number: every placement must resolve here.
var StarterCatalog = map[string]PartSpec{
	"brick-2x4":        {ID: "brick-2x4", Name: "Brick 2 x 4", LDrawID: "3001.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 4, StudsZ: 2, PlatesY: 3, Quantity: 18},
	"brick-2x2":        {ID: "brick-2x2", Name: "Brick 2 x 2", LDrawID: "3003.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 3, Quantity: 16},
	"brick-1x2":        {ID: "brick-1x2", Name: "Brick 1 x 2", LDrawID: "3004.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 1, PlatesY: 3, Quantity: 20},
	"brick-1x1":        {ID: "brick-1x1", Name: "Brick 1 x 1", LDrawID: "3005.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 1, StudsZ: 1, PlatesY: 3, Quantity: 20},
	"plate-2x4":        {ID: "plate-2x4", Name: "Plate 2 x 4", LDrawID: "3020.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 4, StudsZ: 2, PlatesY: 1, Quantity: 12},
	"plate-2x2":        {ID: "plate-2x2", Name: "Plate 2 x 2", LDrawID: "3022.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 1, Quantity: 12},
	"plate-1x2":        {ID: "plate-1x2", Name: "Plate 1 x 2", LDrawID: "3023.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 1, PlatesY: 1, Quantity: 16},
	"slope-2x2":        {ID: "slope-2x2", Name: "Roof Slope 2 x 2", LDrawID: "3039.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 3, Quantity: 8},
	"wheel-holder-2x2": {ID: "wheel-holder-2x2", Name: "Plate 2 x 2 with 2 Wheel Pins", LDrawID: "4600.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 2, StudsZ: 2, PlatesY: 1, Quantity: 4},
	// The shortcut contains a compatible rim and tyre. Its axle origin sits 5
	// LDU below the plate top; the Z offset aligns the round wheel with the
	// holder's stud-row center while occupancy stays on the integer stud grid.
	"wheel": {ID: "wheel", Name: "Wheel Rim 6.4 x 8 with Tyre 8/75 x 8", LDrawID: "4624c04.dat", LDrawStatus: "official", License: "CC-BY-4.0", StudsX: 1, StudsZ: 1, PlatesY: 2, Quantity: 8, OriginYOffsetLDU: 5, OriginCenterZOffsetLDU: 10},
}

var allowedColors = map[int]bool{
	1:  true, // blue
	2:  true, // green
	4:  true, // red
	14: true, // yellow
	15: true, // white
	71: true, // light bluish gray
}
