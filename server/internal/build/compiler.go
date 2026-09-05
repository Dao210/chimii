package build

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// QuestionFor returns at most one high-value question. Prompts that already
// describe a movement or strong silhouette proceed without interruption.
func QuestionFor(prompt string) *ClarifyingQuestion {
	p := strings.ToLower(strings.TrimSpace(prompt))
	if p == "" {
		return &ClarifyingQuestion{ID: "idea", Prompt: "你想创造什么积木朋友？", Options: []string{"会跑的小车", "会飞的动物", "勇敢的机器人"}}
	}
	for _, signal := range []string{"轮", "跑", "车", "飞", "翼", "翅", "尾", "机器人", "robot", "wheel", "fly", "wing"} {
		if strings.Contains(p, signal) {
			return nil
		}
	}
	return &ClarifyingQuestion{
		ID:      "movement",
		Prompt:  "这个新朋友最想怎么动？",
		Options: []string{"用轮子飞快地跑", "张开翅膀飞", "摇尾巴和我打招呼"},
	}
}

// PlanRecipe converts a validated intent into the deliberately small,
// deterministic construction vocabulary. Production generation always gets
// the selected archetype/title/features from the configured LLM first; this
// function never masquerades as an AI fallback when the planner is unavailable.
func PlanRecipe(prompt string, answers map[string]string) AssemblyRecipe {
	joined := strings.ToLower(prompt + " " + answers["idea"] + " " + answers["movement"])
	archetype := "creature"
	features := []string{"friendly-face", "stable-feet"}
	switch {
	case containsAny(joined, "车", "轮", "跑", "car", "wheel", "race"):
		archetype = "racer"
		features = []string{"rolling-base", "driver-cabin"}
	case containsAny(joined, "飞", "翼", "翅", "鸟", "dragon", "fly", "wing"):
		archetype = "flyer"
		features = []string{"wide-wings", "balanced-tail"}
	case containsAny(joined, "机器人", "机械", "robot", "mech"):
		archetype = "robot"
		features = []string{"friendly-face", "strong-arms", "stable-feet"}
	}
	title := map[string]string{"racer": "闪电探险车", "flyer": "云朵飞行兽", "robot": "勇气机器人", "creature": "摇尾巴积木朋友"}[archetype]
	return AssemblyRecipe{Version: 1, Archetype: archetype, Title: title, Prompt: strings.TrimSpace(prompt), Palette: []int{4, 14, 1, 15}, Features: features, Metadata: map[string]string{"planner": "chimii-construction-grammar-v1"}}
}

func containsAny(value string, words ...string) bool {
	for _, word := range words {
		if strings.Contains(value, word) {
			return true
		}
	}
	return false
}

func Compile(recipe AssemblyRecipe, inventory InventorySnapshot, now time.Time) (CompileResult, error) {
	return CompileWithCatalog(recipe, inventory, CatalogVersion, StarterCatalog, now)
}

// CompileWithCatalog is the production compiler entry point. The catalog is
// resolved from the inventory snapshot's immutable catalog version before this
// function is called; the model never supplies part identifiers or geometry.
func CompileWithCatalog(recipe AssemblyRecipe, inventory InventorySnapshot, catalogVersion string, catalog PartCatalog, now time.Time) (CompileResult, error) {
	if len(catalog) == 0 {
		return CompileResult{}, fmt.Errorf("certified part catalog is empty")
	}
	if catalogVersion == "" {
		catalogVersion = inventory.CatalogVersion
	}
	if catalogVersion == "" {
		catalogVersion = CatalogVersion
	}
	placements := placementsFor(recipe)
	placements = resolveCertifiedVariants(placements, recipe, inventory, catalog)
	placements = resolveInventoryColors(placements, inventory)
	var err error
	placements, err = scalePlacementsToTargetWithCatalog(placements, recipe, inventory, catalog)
	if err != nil {
		return CompileResult{Recipe: recipe}, err
	}
	connections := deriveExactConnections(placements, catalog)
	report := validateWithConnections(placements, connections, inventory, catalog)
	plan := BuildPlan{
		Version: 3, KitID: StarterKitID, CatalogVersion: catalogVersion,
		ConnectorSchemaVersion: ConnectorSchemaVersion, PhysicsProfileVersion: PhysicsProfileVersion,
		GeneratorVersion:     GeneratorVersion,
		ModuleLibraryVersion: ModuleLibraryVersion, CompilerVersion: CompilerVersion,
		ValidatorVersion: ValidatorVersion, Title: recipe.Title, Prompt: recipe.Prompt,
		Archetype: recipe.Archetype, Placements: placements, Connections: connections,
		Steps: deriveSteps(placements, report.StepCount), Parts: usedCatalogParts(placements, catalog),
		Validation: report, Inventory: inventory, GeneratedAt: now.UTC(),
	}
	plan.ContentHash = physicalContentHash(plan)
	if !report.Buildable {
		code := BuildErrorStructureInvalid
		for _, issue := range report.Issues {
			if issue.Code == "inventory_exceeded" {
				code = BuildErrorInsufficientInventory
				break
			}
		}
		return CompileResult{Recipe: recipe, Plan: plan}, &BuildError{Code: code, Cause: fmt.Errorf("compiled plan is not buildable: %v", report.Issues)}
	}
	return CompileResult{Recipe: recipe, Plan: plan, MPD: ExportMPD(plan)}, nil
}

// AvailableArchetypes reports which deterministic construction grammars can
// be completed with a frozen inventory. Unlimited mode enables every grammar;
// configured mode accounts for both part shape and per-color quantities.
func AvailableArchetypes(inventory InventorySnapshot) []string {
	return AvailableArchetypesWithCatalog(inventory, StarterCatalog)
}

func AvailableArchetypesWithCatalog(inventory InventorySnapshot, catalog PartCatalog) []string {
	return AvailableArchetypesForRecipe(AssemblyRecipe{}, inventory, catalog)
}

func AvailableArchetypesForRecipe(base AssemblyRecipe, inventory InventorySnapshot, catalog PartCatalog) []string {
	archetypes := []string{"racer", "flyer", "robot", "creature"}
	available := make([]string, 0, len(archetypes))
	for _, archetype := range archetypes {
		recipe := base
		recipe.Archetype = archetype
		placements := resolveCertifiedVariants(placementsFor(recipe), recipe, inventory, catalog)
		placements = resolveInventoryColors(placements, inventory)
		placements, err := scalePlacementsToTargetWithCatalog(placements, recipe, inventory, catalog)
		if err != nil {
			continue
		}
		if ValidateWithCatalog(placements, inventory, catalog).Buildable {
			available = append(available, archetype)
		}
	}
	return available
}

// resolveCertifiedVariants intentionally preserves the reviewed module BOM.
// Replacing one spanning brick with two coplanar bricks preserves occupancy but
// can sever the load path at their seam. Variants must therefore be authored as
// complete, mechanically certified modules rather than inferred per placement.
func resolveCertifiedVariants(placements []Placement, _ AssemblyRecipe, _ InventorySnapshot, _ PartCatalog) []Placement {
	return append([]Placement(nil), placements...)
}

type orientedDimensions struct{ x, z int }

func usedCatalogParts(placements []Placement, catalog PartCatalog) PartCatalog {
	used := make(PartCatalog)
	for _, placement := range placements {
		if part, ok := catalog[placement.PartID]; ok {
			used[placement.PartID] = part
		}
	}
	return used
}

func deriveSteps(placements []Placement, count int) []BuildStep {
	steps := make([]BuildStep, 0, count)
	for number := 1; number <= count; number++ {
		added := make([]string, 0)
		for _, placement := range placements {
			if placement.Step == number {
				added = append(added, placement.ID)
			}
		}
		steps = append(steps, BuildStep{
			Number: number, AddedPlacementIDs: added, CameraPreset: "isometric",
			InstructionKey: "build.step.add_parts",
		})
	}
	return steps
}

func physicalContentHash(plan BuildPlan) string {
	payload := struct {
		KitID                  string      `json:"kit_id"`
		CatalogVersion         string      `json:"catalog_version"`
		ConnectorSchemaVersion int         `json:"connector_schema_version"`
		PhysicsProfileVersion  string      `json:"physics_profile_version"`
		GeneratorVersion       string      `json:"generator_version"`
		ModuleLibraryVersion   string      `json:"module_library_version"`
		CompilerVersion        string      `json:"compiler_version"`
		ValidatorVersion       string      `json:"validator_version"`
		Archetype              string      `json:"archetype"`
		Placements             []Placement `json:"placements"`
	}{
		KitID: plan.KitID, CatalogVersion: plan.CatalogVersion, ConnectorSchemaVersion: plan.ConnectorSchemaVersion,
		PhysicsProfileVersion: plan.PhysicsProfileVersion, GeneratorVersion: plan.GeneratorVersion,
		ModuleLibraryVersion: plan.ModuleLibraryVersion, CompilerVersion: plan.CompilerVersion,
		ValidatorVersion: plan.ValidatorVersion, Archetype: plan.Archetype, Placements: plan.Placements,
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func placementsFor(recipe AssemblyRecipe) []Placement {
	var p []Placement
	add := func(part string, color, x, y, z, rotation, step int, module string) {
		p = append(p, Placement{ID: fmt.Sprintf("p%02d", len(p)+1), PartID: part, Color: color, X: x, Y: y, Z: z, Rotation: rotation, Step: step, Module: module})
	}

	switch recipe.Archetype {
	case "racer":
		// 4600.dat provides real wheel pins; 4624c04.dat is the matching
		// rim+tyre shortcut. Build both holders and their bridge before adding
		// wheels so every saved step has a real support polygon.
		for _, z := range []int{0, 2} {
			add("wheel-holder-2x2", 71, 0, 0, z, 0, 1, "rolling-base")
		}
		add("plate-2x4", 4, 0, 1, 0, 90, 2, "rolling-base")
		for _, z := range []int{0, 2} {
			add("wheel", 71, -1, 0, z, 90, 3, "wheels")
			add("wheel", 71, 2, 0, z, 270, 3, "wheels")
		}
		add("brick-2x4", 14, 0, 2, 0, 90, 4, "body")
		add("brick-2x2", 1, 0, 5, 1, 0, 5, "driver-cabin")
		add("slope-2x2", 15, 0, 8, 1, 0, 6, "driver-cabin")
	case "flyer":
		add("brick-2x4", 1, 0, 0, 0, 0, 1, "body")
		add("brick-2x2", 14, 1, 3, 0, 0, 2, "head")
		add("plate-2x4", 4, -3, 3, 0, 0, 3, "left-wing")
		add("plate-2x4", 4, 3, 3, 0, 0, 3, "right-wing")
		add("plate-1x2", 14, 1, 6, 1, 0, 4, "tail")
		add("slope-2x2", 15, 1, 7, 1, 0, 5, "tail")
	case "robot":
		p = append(p, robotPlacements(recipe)...)
	default:
		add("brick-2x4", 2, 0, 0, 0, 0, 1, "body")
		add("brick-2x2", 14, 0, 3, 0, 0, 2, "head")
		add("brick-1x1", 15, 0, 6, 0, 0, 3, "left-eye")
		add("brick-1x1", 15, 1, 6, 0, 0, 3, "right-eye")
		add("plate-1x2", 4, 3, 3, 0, 0, 4, "tail")
		add("slope-2x2", 4, 3, 4, 0, 0, 5, "tail-tip")
	}
	return p
}

func robotPlacements(recipe AssemblyRecipe) []Placement {
	paletteColor := func(index int, fallback int) int {
		if index < len(recipe.Palette) && IsAllowedColor(recipe.Palette[index]) {
			return recipe.Palette[index]
		}
		return fallback
	}
	primaryColor := paletteColor(0, 4)
	coreColor := paletteColor(1, 14)
	accentColor := paletteColor(2, 15)
	headColor := paletteColor(3, 1)

	parts := make([]Placement, 0, 12)
	add := func(part string, color, x, y, z, rotation, step int, module string) {
		parts = append(parts, Placement{ID: fmt.Sprintf("r%02d", len(parts)+1), PartID: part, Color: color, X: x, Y: y, Z: z, Rotation: rotation, Step: step, Module: module})
	}

	add("brick-2x2", primaryColor, 0, 0, 0, 0, 1, "left-foot")
	add("brick-2x2", primaryColor, 2, 0, 0, 0, 1, "right-foot")
	add("brick-2x4", coreColor, 0, 3, 0, 0, 2, "body")
	add("brick-2x4", coreColor, 0, 6, 0, 0, 3, "body")
	add("brick-2x2", headColor, 1, 9, 0, 0, 4, "head")
	add("brick-1x1", accentColor, 1, 12, 0, 0, 5, "left-eye")
	add("brick-1x1", accentColor, 2, 12, 0, 0, 5, "right-eye")

	return parts
}

func Validate(placements []Placement, inventory InventorySnapshot) ValidationReport {
	return ValidateWithCatalog(placements, inventory, StarterCatalog)
}

func ValidateWithCatalog(placements []Placement, inventory InventorySnapshot, catalog PartCatalog) ValidationReport {
	return validateWithConnections(placements, deriveExactConnections(placements, catalog), inventory, catalog)
}

func validateWithConnections(placements []Placement, connections []Connection, inventory InventorySnapshot, catalog PartCatalog) ValidationReport {
	report := ValidationReport{
		Buildable:       true,
		Issues:          make([]ValidationIssue, 0),
		UsedParts:       map[string]int{},
		PartCount:       len(placements),
		ConnectionCount: len(connections),
	}
	if len(placements) == 0 {
		report.Issues = append(report.Issues, ValidationIssue{Code: "empty_build", Message: "搭建方案中没有积木块"})
	}
	maxStep := 0
	occupied := map[[3]int]string{}
	usedInventory := map[inventoryKey]int{}
	availableInventory := inventory.quantities()
	placementsByID := make(map[string]Placement, len(placements))
	certificationByPartID := make(map[string]bool)
	for _, p := range placements {
		if _, exists := placementsByID[p.ID]; exists || p.ID == "" {
			report.Issues = append(report.Issues, ValidationIssue{Code: "duplicate_placement", Message: "积木块标识为空或重复", PlacementID: p.ID})
		}
		placementsByID[p.ID] = p
		spec, ok := catalog[p.PartID]
		if !ok {
			report.Issues = append(report.Issues, ValidationIssue{Code: "unknown_part", Message: "零件不在套装目录中", PlacementID: p.ID})
			continue
		}
		certified, checked := certificationByPartID[p.PartID]
		if !checked {
			certified = partIsMechanicallyCertified(spec)
			certificationByPartID[p.PartID] = certified
		}
		if !certified {
			report.Issues = append(report.Issues, ValidationIssue{Code: "uncertified_part", Message: "零件缺少已认证的连接器或占位语义", PlacementID: p.ID})
		}
		if !allowedColors[p.Color] {
			report.Issues = append(report.Issues, ValidationIssue{Code: "unknown_color", Message: "颜色不在套装目录中", PlacementID: p.ID})
		}
		report.UsedParts[p.PartID]++
		key := inventoryKey{partID: p.PartID, color: p.Color}
		usedInventory[key]++
		if inventory.Configured && usedInventory[key] > availableInventory[key] {
			report.Issues = append(report.Issues, ValidationIssue{Code: "inventory_exceeded", Message: "使用数量超过我的积木块库存", PlacementID: p.ID})
		}
		size := orientedSizeWith(p, catalog)
		if p.Y < 0 || p.X < -16 || p.X+size.x > 17 || p.Z < -16 || p.Z+size.z > 17 {
			report.Issues = append(report.Issues, ValidationIssue{Code: "out_of_bounds", Message: "零件超出安全搭建范围", PlacementID: p.ID})
		}
		if p.Rotation%90 != 0 {
			report.Issues = append(report.Issues, ValidationIssue{Code: "invalid_rotation", Message: "零件角度必须按 90° 旋转", PlacementID: p.ID})
		}
		if p.Step <= 0 {
			report.Issues = append(report.Issues, ValidationIssue{Code: "invalid_step", Message: "积木块必须属于有效搭建步骤", PlacementID: p.ID})
		}
		if p.Step > maxStep {
			maxStep = p.Step
		}
		for x := p.X; x < p.X+size.x; x++ {
			for z := p.Z; z < p.Z+size.z; z++ {
				for y := p.Y; y < p.Y+spec.PlatesY; y++ {
					cell := [3]int{x, y, z}
					if previous, exists := occupied[cell]; exists {
						report.Issues = append(report.Issues, ValidationIssue{Code: "collision", Message: "零件与 " + previous + " 重叠", PlacementID: p.ID})
					}
					occupied[cell] = p.ID
				}
			}
		}
	}

	// Every elevated part must engage exact certified connectors from an earlier
	// step. Larger parts require at least two studs so an incidental one-stud
	// overlap cannot masquerade as a robust attachment.
	for _, placement := range placements {
		if placement.Y == 0 {
			continue
		}
		engaged := supportEngagement(placement, placementsByID, connections, catalog)
		if engaged == 0 {
			report.Issues = append(report.Issues, ValidationIssue{Code: "unsupported", Message: "零件下方没有精确匹配的连接器", PlacementID: placement.ID})
		} else if engaged < minimumSupportEngagement(placement, catalog) {
			report.Issues = append(report.Issues, ValidationIssue{Code: "weak_connection", Message: "零件与主体的咬合点不足", PlacementID: placement.ID})
		}
	}

	// Connectivity is evaluated from certified connector pairs, never bounding
	// box proximity.
	if len(placements) > 0 {
		adjacency := make(map[string][]string, len(placements))
		for _, connection := range connections {
			adjacency[connection.APlacementID] = append(adjacency[connection.APlacementID], connection.BPlacementID)
			adjacency[connection.BPlacementID] = append(adjacency[connection.BPlacementID], connection.APlacementID)
		}
		visited := map[string]bool{placements[0].ID: true}
		queue := []string{placements[0].ID}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, neighbor := range adjacency[current] {
				if !visited[neighbor] {
					visited[neighbor] = true
					queue = append(queue, neighbor)
				}
			}
		}
		for _, placement := range placements {
			if !visited[placement.ID] {
				report.Issues = append(report.Issues, ValidationIssue{Code: "disconnected", Message: "零件没有通过认证连接器连接到主体", PlacementID: placement.ID})
			}
		}
	}

	stabilityIssues, minimumMargin := validateStepStability(placements, connections, catalog, maxStep)
	report.Issues = append(report.Issues, stabilityIssues...)
	report.MinimumStabilityMarginMils = minimumMargin
	report.StepCount = maxStep
	report.Buildable = len(report.Issues) == 0
	sort.Slice(report.Issues, func(i, j int) bool {
		if report.Issues[i].Code != report.Issues[j].Code {
			return report.Issues[i].Code < report.Issues[j].Code
		}
		return report.Issues[i].PlacementID < report.Issues[j].PlacementID
	})
	return report
}

func orientedSizeWith(placement Placement, catalog PartCatalog) orientedDimensions {
	spec := catalog[placement.PartID]
	x, z := spec.StudsX, spec.StudsZ
	if placement.Rotation%180 != 0 {
		x, z = z, x
	}
	return orientedDimensions{x: x, z: z}
}

func topYWith(placement Placement, catalog PartCatalog) int {
	return placement.Y + catalog[placement.PartID].PlatesY
}
