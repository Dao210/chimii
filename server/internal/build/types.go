package build

import "time"

// PartSpec is the server-owned catalog entry for a physical part in the
// CHIMII Starter Kit. Dimensions are expressed in studs horizontally and
// plates vertically; LDraw coordinates are derived only at export time.
type PartSpec struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	Category               string          `json:"category"`
	PopularityRank         int             `json:"popularity_rank,omitempty"`
	CertificationLevel     string          `json:"certification_level,omitempty"`
	AutoBuildEligible      bool            `json:"auto_build_eligible,omitempty"`
	GeometryProfile        string          `json:"geometry_profile,omitempty"`
	LDrawID                string          `json:"ldraw_id"`
	LDrawStatus            string          `json:"ldraw_status"`
	License                string          `json:"license"`
	StudsX                 int             `json:"studs_x"`
	StudsZ                 int             `json:"studs_z"`
	PlatesY                int             `json:"plates_y"`
	Quantity               int             `json:"quantity"`
	HasTopStuds            bool            `json:"has_top_studs,omitempty"`
	HasBottomReceptors     bool            `json:"has_bottom_receptors,omitempty"`
	OriginYOffsetLDU       int             `json:"origin_y_offset_ldu,omitempty"`
	OriginCenterZOffsetLDU int             `json:"origin_center_z_offset_ldu,omitempty"`
	Bounds                 PartBounds      `json:"bounds,omitempty"`
	Connectors             []PartConnector `json:"connectors,omitempty"`
	Occupancy              PartOccupancy   `json:"occupancy,omitempty"`
}

// PartBounds preserves the catalog-derived render bounds. Runtime collision
// checks use the certified occupancy profile; bounds are retained for audit and
// future special-part profiles.
type PartBounds struct {
	MinX float64 `json:"min_x,omitempty"`
	MinY float64 `json:"min_y,omitempty"`
	MinZ float64 `json:"min_z,omitempty"`
	MaxX float64 `json:"max_x,omitempty"`
	MaxY float64 `json:"max_y,omitempty"`
	MaxZ float64 `json:"max_z,omitempty"`
}

// PartConnector is a catalog-owned mating point in local LDraw coordinates.
// IDs are stable within one immutable catalog revision.
type PartConnector struct {
	ID            string `json:"id,omitempty"`
	Kind          string `json:"kind"`
	X             int    `json:"x_ldu"`
	Y             int    `json:"y_ldu"`
	Z             int    `json:"z_ldu"`
	Direction     string `json:"direction"`
	CapacityUnits int    `json:"capacity_units,omitempty"`
}

// PartOccupancy is the conservative collision profile used by the online
// validator. The online profile supports rectangular volumes plus reviewed
// ground-contact behavior; other special parts remain ineligible for auto-build.
type PartOccupancy struct {
	Profile              string `json:"profile,omitempty"`
	GroundContactProfile string `json:"ground_contact_profile,omitempty"`
	StudsX               int    `json:"studs_x,omitempty"`
	StudsZ               int    `json:"studs_z,omitempty"`
	PlatesY              int    `json:"plates_y,omitempty"`
}

// PartCatalog is a version-frozen set of certified part semantics. BuildPlan
// stores only the subset it actually uses, so old creations stay replayable
// after a newer catalog release becomes active.
type PartCatalog map[string]PartSpec

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
	ID            string   `json:"id"`
	APlacementID  string   `json:"a_placement_id"`
	BPlacementID  string   `json:"b_placement_id"`
	AConnectorIDs []string `json:"a_connector_ids,omitempty"`
	BConnectorIDs []string `json:"b_connector_ids,omitempty"`
	Kind          string   `json:"kind"`
	EngagedCount  int      `json:"engaged_count,omitempty"`
	CapacityUnits int      `json:"capacity_units,omitempty"`
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
	Buildable                  bool              `json:"buildable"`
	Issues                     []ValidationIssue `json:"issues"`
	PartCount                  int               `json:"part_count"`
	StepCount                  int               `json:"step_count"`
	ConnectionCount            int               `json:"connection_count,omitempty"`
	MinimumStabilityMarginMils int               `json:"minimum_stability_margin_mils,omitempty"`
	UsedParts                  map[string]int    `json:"used_parts"`
}

type BuildPlan struct {
	Version                int                 `json:"version"`
	KitID                  string              `json:"kit_id"`
	CatalogVersion         string              `json:"catalog_version"`
	ConnectorSchemaVersion int                 `json:"connector_schema_version,omitempty"`
	PhysicsProfileVersion  string              `json:"physics_profile_version,omitempty"`
	GeneratorVersion       string              `json:"generator_version,omitempty"`
	ModuleLibraryVersion   string              `json:"module_library_version"`
	CompilerVersion        string              `json:"compiler_version"`
	ValidatorVersion       string              `json:"validator_version"`
	Title                  string              `json:"title"`
	Prompt                 string              `json:"prompt"`
	Archetype              string              `json:"archetype"`
	Placements             []Placement         `json:"placements"`
	Connections            []Connection        `json:"connections"`
	Steps                  []BuildStep         `json:"steps"`
	Parts                  map[string]PartSpec `json:"parts"`
	Validation             ValidationReport    `json:"validation"`
	Inventory              InventorySnapshot   `json:"inventory"`
	ContentHash            string              `json:"content_hash"`
	GeneratedAt            time.Time           `json:"generated_at"`
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
