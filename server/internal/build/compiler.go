package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

func Compile(recipe AssemblyRecipe, inventory InventorySnapshot, now time.Time) (CompileResult, error) {
	return CompileWithCatalog(recipe, inventory, CatalogVersion, StarterCatalog, now)
}

// CompileWithCatalog is the production compiler entry point. The catalog is
// resolved from the inventory snapshot's immutable catalog version before this
// function is called; the model never supplies part identifiers or geometry.
// CompileWithCatalog tries only explicitly permitted layout alternatives, within
// a fixed budget. It never changes required colors, part counts, or module kinds.
func CompileWithCatalog(recipe AssemblyRecipe, inventory InventorySnapshot, catalogVersion string, catalog PartCatalog, now time.Time) (CompileResult, error) {
	if recipe.Design != nil {
		return CompileDesign(context.Background(), recipe, inventory, catalogVersion, catalog, now)
	}
	result, err := compileCandidate(recipe, inventory, catalogVersion, catalog, now)
	if code, ok := BuildErrorCode(err); !ok || code != BuildErrorStructureInvalid {
		return result, err
	}
	attempts := 0
	for i, m := range recipe.Modules {
		for _, port := range m.AlternativePorts {
			if port == m.Port {
				continue
			}
			attempts++
			if attempts > 4 {
				return result, err
			}
			candidate := recipe
			candidate.Modules = append([]ModuleInstance(nil), recipe.Modules...)
			candidate.Modules[i].Port = port
			next, nextErr := compileCandidate(candidate, inventory, catalogVersion, catalog, now)
			if nextErr == nil {
				return next, nil
			}
		}
	}
	return result, err
}

func compileCandidate(recipe AssemblyRecipe, inventory InventorySnapshot, catalogVersion string, catalog PartCatalog, now time.Time) (CompileResult, error) {
	if len(catalog) == 0 {
		return CompileResult{}, fmt.Errorf("certified part catalog is empty")
	}
	if catalogVersion == "" {
		catalogVersion = inventory.CatalogVersion
	}
	if catalogVersion == "" {
		catalogVersion = CatalogVersion
	}
	placements, err := ExpandRecipe(recipe)
	if err != nil {
		return CompileResult{}, err
	}
	if !recipe.Constraints.ExactColors {
		placements = resolveInventoryColors(placements, inventory)
	}
	placements, err = scalePlacementsToTargetWithCatalog(placements, recipe, inventory, catalog)
	if err != nil {
		return CompileResult{Recipe: recipe}, err
	}
	return compilePlacements(recipe, placements, inventory, catalogVersion, catalog, now)
}

func compilePlacements(recipe AssemblyRecipe, placements []Placement, inventory InventorySnapshot, catalogVersion string, catalog PartCatalog, now time.Time) (CompileResult, error) {
	if recipe.Modules == nil {
		recipe.Modules = []ModuleInstance{}
	}
	if recipe.Palette == nil {
		recipe.Palette = []int{}
	}
	if recipe.Features == nil {
		recipe.Features = []string{}
	}
	if recipe.Requirements == nil {
		recipe.Requirements = []string{}
	}
	if recipe.Metadata == nil {
		recipe.Metadata = map[string]string{}
	}
	if catalogVersion == "" {
		catalogVersion = inventory.CatalogVersion
	}
	if catalogVersion == "" {
		catalogVersion = CatalogVersion
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
		Validation: report, GeneratedAt: now.UTC(),
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
