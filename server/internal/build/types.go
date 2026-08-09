package build

import "time"

// PartSpec is the server-owned catalog entry for a physical part in the
// CHIMII Starter Kit. Dimensions are expressed in studs horizontally and
// plates vertically; LDraw coordinates are derived only at export time.
type PartSpec struct {
	ID                     string `json:"id"`
	Name                   string `json:"name"`
	Category               string `json:"category"`
	LDrawID                string `json:"ldraw_id"`
	LDrawStatus            string `json:"ldraw_status"`
	License                string `json:"license"`
	StudsX                 int    `json:"studs_x"`
	StudsZ                 int    `json:"studs_z"`
	PlatesY                int    `json:"plates_y"`
	Quantity               int    `json:"quantity"`
	OriginYOffsetLDU       int    `json:"origin_y_offset_ldu,omitempty"`
	OriginCenterZOffsetLDU int    `json:"origin_center_z_offset_ldu,omitempty"`
}

// InventoryItem is one user-owned part/color combination. Quantity is an
// upper bound for a single generated model, not a permanently consumed count.
type InventoryItem struct {
	PartID   string `json:"part_id"`
	Color    int    `json:"color"`
	Quantity int    `json:"quantity"`
}

// InventorySnapshot freezes the physical inventory semantics for one build.
// Configured=false deliberately means every catalog part/color is available
// without a quantity limit.
type InventorySnapshot struct {
	Configured     bool            `json:"configured"`
	CatalogVersion string          `json:"catalog_version"`
	Revision       int32           `json:"revision"`
	Items          []InventoryItem `json:"items"`
	ContentHash    string          `json:"content_hash"`
}

type Placement struct {
	ID       string `json:"id"`
	PartID   string `json:"part_id"`
	Color    int    `json:"color"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Z        int    `json:"z"`
	Rotation int    `json:"rotation"`
	Step     int    `json:"step"`
	Module   string `json:"module"`
}

type Connection struct {
	ID           string `json:"id"`
	APlacementID string `json:"a_placement_id"`
	BPlacementID string `json:"b_placement_id"`
	Kind         string `json:"kind"`
}

type BuildStep struct {
	Number            int      `json:"number"`
	AddedPlacementIDs []string `json:"added_placement_ids"`
	CameraPreset      string   `json:"camera_preset"`
	InstructionKey    string   `json:"instruction_key"`
}

type AssemblyRecipe struct {
	Version   int               `json:"version"`
	Archetype string            `json:"archetype"`
	Title     string            `json:"title"`
	Prompt    string            `json:"prompt"`
	Palette   []int             `json:"palette"`
	Features  []string          `json:"features"`
	Metadata  map[string]string `json:"metadata"`
}

type ValidationIssue struct {
	Code        string `json:"code"`
	Message     string `json:"message"`
	PlacementID string `json:"placement_id,omitempty"`
}

type ValidationReport struct {
	Buildable bool              `json:"buildable"`
	Issues    []ValidationIssue `json:"issues"`
	PartCount int               `json:"part_count"`
	StepCount int               `json:"step_count"`
	UsedParts map[string]int    `json:"used_parts"`
}

type BuildPlan struct {
	Version              int                 `json:"version"`
	KitID                string              `json:"kit_id"`
	CatalogVersion       string              `json:"catalog_version"`
	ModuleLibraryVersion string              `json:"module_library_version"`
	CompilerVersion      string              `json:"compiler_version"`
	ValidatorVersion     string              `json:"validator_version"`
	Title                string              `json:"title"`
	Prompt               string              `json:"prompt"`
	Archetype            string              `json:"archetype"`
	Placements           []Placement         `json:"placements"`
	Connections          []Connection        `json:"connections"`
	Steps                []BuildStep         `json:"steps"`
	Parts                map[string]PartSpec `json:"parts"`
	Validation           ValidationReport    `json:"validation"`
	Inventory            InventorySnapshot   `json:"inventory"`
	ContentHash          string              `json:"content_hash"`
	GeneratedAt          time.Time           `json:"generated_at"`
}

type ClarifyingQuestion struct {
	ID      string   `json:"id"`
	Prompt  string   `json:"prompt"`
	Options []string `json:"options"`
}

type CompileResult struct {
	Recipe AssemblyRecipe `json:"recipe"`
	Plan   BuildPlan      `json:"plan"`
	MPD    string         `json:"mpd"`
}
