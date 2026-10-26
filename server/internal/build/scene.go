package build

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	SceneVersion          = 1
	SceneGeneratorVersion = "construction-scene-v3"
	MaxSceneParts         = 1200
	MaxSceneObjects       = 8
	MaxSceneModules       = 64
	maxSceneCells         = 32768
	maxObjectParts        = 800
	maxSceneSearchNodes   = 96000
)

// SceneSpec describes independently grounded objects built from reusable
// constructions. Part IDs, connector geometry and placements remain server owned.
type SceneSpec struct {
	Version int           `json:"version"`
	Brief   string        `json:"brief"`
	Objects []SceneObject `json:"objects"`
}

type SceneObject struct {
	ID         string              `json:"id"`
	Label      string              `json:"label"`
	Family     string              `json:"family"`
	Position   DesignVector        `json:"position"`
	Rotation   int                 `json:"rotation"`
	Modules    []Construction      `json:"modules"`
	Interfaces []AssemblyInterface `json:"interfaces,omitempty"`
}

type Construction struct {
	ID              string                  `json:"id"`
	Label           string                  `json:"label"`
	Kind            string                  `json:"kind"`
	Role            string                  `json:"role,omitempty"`
	Parent          string                  `json:"parent,omitempty"`
	Attach          *ConstructionAttachment `json:"attach,omitempty"`
	Position        DesignVector            `json:"position"`
	Size            DesignVector            `json:"size"`
	Rotation        int                     `json:"rotation"`
	Color           int                     `json:"color"`
	Openings        []WallOpening           `json:"openings,omitempty"`
	CeilingOpenings []DeckOpening           `json:"ceiling_openings,omitempty"`
	Design          *DesignSpec             `json:"design,omitempty"`
	Requirements    []int                   `json:"requirements,omitempty"`
}

type ConstructionAttachment struct {
	Parent  string       `json:"parent,omitempty"`
	Parents []string     `json:"parents,omitempty"`
	Anchor  string       `json:"anchor"`
	Offset  DesignVector `json:"offset"`
}

type WallOpening struct {
	Face   string `json:"face,omitempty"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type DeckOpening struct {
	X     int `json:"x"`
	Z     int `json:"z"`
	Width int `json:"width"`
	Depth int `json:"depth"`
}

type RequirementEvidence struct {
	Requirement string   `json:"requirement"`
	Modules     []string `json:"modules"`
}

// Geometry checks describe their actual profile. Representation of a textual
// requirement is evidence of attribution, not proof of visual/semantic fidelity.
type BuildEvidence struct {
	Objects      int                   `json:"objects"`
	Modules      int                   `json:"modules"`
	Geometry     string                `json:"geometry"`
	Steps        string                `json:"steps"`
	Semantic     string                `json:"semantic"`
	Physical     string                `json:"physical"`
	RetailColors string                `json:"retail_colors"`
	Requirements []RequirementEvidence `json:"requirements"`
	Review       string                `json:"review,omitempty"`
	Interfaces   []InterfaceEvidence   `json:"interfaces,omitempty"`
	LoadEstimate *StaticLoadEstimate   `json:"load_estimate,omitempty"`
	Visual       *VisualReviewEvidence `json:"visual,omitempty"`
}

func sceneID(id string) bool {
	if id == "" || len(id) > 48 {
		return false
	}
	for _, r := range id {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// ValidateScene performs bounded input validation without searching for parts.
func ValidateScene(recipe AssemblyRecipe) error {
	s := recipe.Scene
	if recipe.Version != 4 || s == nil || (s.Version != SceneVersion && s.Version != 2) || recipe.Design != nil || len(recipe.Modules) != 0 ||
		len(s.Objects) == 0 || len(s.Objects) > MaxSceneObjects || strings.TrimSpace(s.Brief) == "" || len([]rune(s.Brief)) > 600 ||
		recipe.Constraints.PartCount < 0 || recipe.Constraints.PartCount > MaxSceneParts || len(recipe.Constraints.RequiredModules) > 0 {
		return designError("invalid scene recipe")
	}
	ids := map[string]bool{}
	if err := validateAssemblyGraph(*s); err != nil {
		return err
	}
	covered := map[int]bool{}
	problems := []string{}
	modules, cells := 0, 0
	for _, object := range s.Objects {
		if !sceneID(object.ID) || ids[object.ID] || strings.TrimSpace(object.Label) == "" || len([]rune(object.Label)) > 80 ||
			object.Position.Y != 0 || abs(object.Position.X) > 48 || abs(object.Position.Z) > 48 || object.Rotation < 0 || object.Rotation > 270 || object.Rotation%90 != 0 || len(object.Modules) == 0 {
			problems = append(problems, fmt.Sprintf("invalid object %q: ID must be unique ASCII, label required, position=(%d,%d,%d) needs X/Z -48..48 and Y=0, rotation=%d needs 0/90/180/270, modules must be nonempty", object.ID, object.Position.X, object.Position.Y, object.Position.Z, object.Rotation))
		}
		ids[object.ID] = true
		switch object.Family {
		case "architecture", "vehicle", "spacecraft", "creature", "figure", "landscape":
		default:
			problems = append(problems, "unknown design family: "+object.Family)
		}
		seen := map[string]bool{}
		for _, m := range object.Modules {
			modules++
			if modules > MaxSceneModules {
				return designError("more than 64 scene constructions")
			}
			if !sceneID(m.ID) || seen[m.ID] || strings.TrimSpace(m.Label) == "" || len([]rune(m.Label)) > 80 ||
				m.Rotation < 0 || m.Rotation > 270 || m.Rotation%90 != 0 || !IsAllowedColor(m.Color) ||
				abs(m.Position.X) > 32 || abs(m.Position.Z) > 32 || m.Position.Y < 0 || m.Position.Y > 96 {
				problems = append(problems, fmt.Sprintf("invalid construction %s/%s: label=%q (required, <=80), position=(%d,%d,%d) (X/Z +-32,Y 0..96), rotation=%d (0/90/180/270), color=%d (allowed %v), ID unique ASCII <=48", object.ID, m.ID, m.Label, m.Position.X, m.Position.Y, m.Position.Z, m.Rotation, m.Color, AllowedColorCodes()))
			}
			seen[m.ID] = true
			if m.Attach != nil {
				a := m.Attach
				valid := false
				if a.Anchor == "ground" {
					valid = m.Kind != "paint" && a.Parent == "" && len(a.Parents) == 0 && a.Offset.Y == 0
				} else if m.Kind == "paint" {
					valid = sceneID(a.Parent) && len(a.Parents) == 0 && (a.Anchor == "top" || a.Anchor == "front" || a.Anchor == "back" || a.Anchor == "left" || a.Anchor == "right" || a.Anchor == "inside")
					if m.Parent != "" && m.Parent != a.Parent {
						valid = false
					}
					m.Parent = a.Parent
				} else if a.Anchor == "top" && a.Offset.Y == 0 {
					valid = sceneID(a.Parent) && len(a.Parents) == 0
					if a.Parent == "" && len(a.Parents) >= 2 && len(a.Parents) <= 8 {
						valid = true
						seenParents := map[string]bool{}
						for _, id := range a.Parents {
							if !sceneID(id) || seenParents[id] {
								valid = false
							}
							seenParents[id] = true
						}
					}
				}
				if !valid {
					problems = append(problems, fmt.Sprintf("invalid attachment %s/%s: kind=%s anchor=%s parent=%s parents=%v offset=(%d,%d,%d). Normal constructions require ground without parents and Y=0, or top with valid parent(s) and Y offset=0. Side faces front/back/left/right/inside require embedded paint, not side connectors.", object.ID, m.ID, m.Kind, a.Anchor, a.Parent, a.Parents, a.Offset.X, a.Offset.Y, a.Offset.Z))
				}
				if abs(a.Offset.X) > 32 || abs(a.Offset.Z) > 32 || abs(a.Offset.Y) > 96 {
					problems = append(problems, "attachment offset outside bounds: "+m.ID)
				}
			}
			seenRequirements := map[int]bool{}
			for _, n := range m.Requirements {
				if seenRequirements[n] {
					problems = append(problems, "duplicate requirement attribution: "+object.ID+"/"+m.ID)
					continue
				}
				seenRequirements[n] = true
				if n < 0 || n >= len(recipe.Requirements) {
					problems = append(problems, "unknown requirement index: "+object.ID+"/"+m.ID)
					continue
				}
				covered[n] = true
			}
			local, _, err := constructionCells(m)
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s/%s: %s", object.ID, m.ID, err))
				continue
			}
			cells += len(local)
			if cells > maxSceneCells {
				return designError("scene volume budget exceeded")
			}
		}
		for _, m := range object.Modules {
			if m.Attach == nil {
				continue
			}
			parents := append([]string{}, m.Attach.Parents...)
			if m.Attach.Parent != "" {
				parents = append(parents, m.Attach.Parent)
			}
			for _, id := range parents {
				if id == m.ID || !seen[id] {
					problems = append(problems, fmt.Sprintf("attachment %s/%s requires a different parent %q in the SAME object; keep a display stand and its supported model in one object", object.ID, m.ID, id))
				}
			}
		}
		if err := checkSceneAttachmentGraph(object); err != nil {
			problems = append(problems, err.Error())
		}
	}
	for n := range recipe.Requirements {
		colors := map[int]bool{}
		stairs, windows, wheels := false, false, 0
		constructions := 0
		for _, object := range recipe.Scene.Objects {
			for _, module := range object.Modules {
				for _, index := range module.Requirements {
					if index == n {
						if module.Kind != "paint" {
							constructions++
						}
						stairs = stairs || module.Kind == "stairs"
						windows = windows || (module.Kind == "wall" || module.Kind == "frame" || module.Kind == "storey") && len(module.Openings) > 0
						if module.Kind == "wheelbase" {
							wheels += 4
						}
						if module.Kind != "shape" {
							colors[module.Color] = true
						}
						if module.Kind == "shape" && module.Design != nil {
							for _, shape := range module.Design.Shapes {
								if shape.Operation == "add" {
									colors[shape.Color] = true
								}
							}
						}
					}
				}
			}
		}
		if err := checkSceneRequirementFunction(recipe.Requirements[n], stairs, windows, wheels); err != nil {
			problems = append(problems, fmt.Sprintf("requirement %d: %s", n, err))
		}
		if err := checkScenePairedRequirement(recipe.Requirements[n], constructions); err != nil {
			problems = append(problems, fmt.Sprintf("requirement %d: %s", n, err))
		}
		for _, color := range RequiredSceneColors(recipe.Requirements[n]) {
			if !colors[color] {
				problems = append(problems, fmt.Sprintf("requirement %d (%s) needs %s (LDraw color %d) in its attributed constructions; choose the required color, never substitute an unrelated patch", n, recipe.Requirements[n], SceneColorNames()[color], color))
			}
		}
		if !covered[n] {
			problems = append(problems, fmt.Sprintf("requirement %d has no construction evidence", n))
		}
	}
	if len(problems) > 0 {
		return designError(strings.Join(problems[:min(len(problems), 16)], "\n"))
	}
	return nil
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func transformCell(p DesignVector, origin DesignVector, rotation int) DesignVector {
	x, z := p.X, p.Z
	switch rotation {
	case 90:
		x, z = -z-1, x
	case 180:
		x, z = -x-1, -z-1
	case 270:
		x, z = z, -x-1
	}
	return DesignVector{origin.X + x, origin.Y + p.Y, origin.Z + z}
}

func sceneObjectTarget(object SceneObject, catalog PartCatalog) (designTarget, []Placement, error) {
	target := designTarget{}
	fixed := []Placement{}
	collisions := map[string]string{}
	for _, m := range object.Modules {
		if m.Kind == "paint" {
			continue
		}
		local, special, err := constructionCells(m)
		if err != nil {
			return nil, nil, err
		}
		var shape designTarget
		if m.Kind == "shape" {
			shape, err = RasterizeDesign(*m.Design)
			if err != nil {
				return nil, nil, err
			}
		}
		for p := range local {
			world := transformCell(transformCell(p, m.Position, m.Rotation), object.Position, object.Rotation)
			if world.X < -64 || world.X >= 64 || world.Z < -64 || world.Z >= 64 || world.Y < 0 || world.Y >= 128 {
				return nil, nil, designError("construction outside scene bounds")
			}
			if previous, exists := target[world]; exists {
				key := previous.Node + "|" + object.ID + "/" + m.ID
				if _, seen := collisions[key]; !seen {
					collisions[key] = fmt.Sprintf("overlap %s and %s/%s at (%d,%d,%d)", previous.Node, object.ID, m.ID, world.X, world.Y, world.Z)
				}
				continue
			}
			cell := targetCell{object.ID + "/" + m.ID, m.Color}
			if shape != nil {
				cell.Color = shape[p].Color
				cell.Node += "/" + shape[p].Node
			}
			target[world] = cell
		}
		for _, p := range special {
			spec, ok := catalog[p.PartID]
			if !ok || !partIsMechanicallyCertified(spec) {
				return nil, nil, &BuildError{Code: BuildErrorUnsupported, Cause: fmt.Errorf("construction requires reviewed part %s", p.PartID)}
			}
			// Special constructions are upright, catalog-backed assemblies. Rotate
			// their occupancy corners with exactly the same scene transform.
			size := orientedSizeWith(p, catalog)
			minX, minZ := 1000, 1000
			for _, corner := range []DesignVector{{p.X, p.Y, p.Z}, {p.X + size.x - 1, p.Y, p.Z}, {p.X, p.Y, p.Z + size.z - 1}, {p.X + size.x - 1, p.Y, p.Z + size.z - 1}} {
				q := transformCell(transformCell(corner, m.Position, m.Rotation), object.Position, object.Rotation)
				minX, minZ = min(minX, q.X), min(minZ, q.Z)
			}
			p.X, p.Y, p.Z = minX, p.Y+m.Position.Y, minZ
			p.Rotation = (p.Rotation + m.Rotation + object.Rotation) % 360
			p.ID, p.Module, p.Object, p.Color = object.ID+"/"+m.ID+"/"+p.ID, object.ID+"/"+m.ID, object.ID, m.Color
			fixed = append(fixed, p)
		}
	}
	if len(collisions) > 0 {
		keys := []string{}
		for key := range collisions {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		messages := []string{}
		for _, key := range keys[:min(len(keys), 16)] {
			messages = append(messages, collisions[key])
		}
		return nil, nil, designError(strings.Join(messages, "; ") + "; separate all implicated construction volumes")
	}
	for _, m := range object.Modules {
		if m.Kind != "paint" {
			continue
		}
		local, _, err := constructionCells(m)
		if err != nil {
			return nil, nil, err
		}
		for p := range local {
			world := transformCell(transformCell(p, m.Position, m.Rotation), object.Position, object.Rotation)
			previous, exists := target[world]
			if !exists || !strings.HasPrefix(previous.Node, object.ID+"/"+m.Parent) || previous.Node != object.ID+"/"+m.Parent && !strings.HasPrefix(previous.Node, object.ID+"/"+m.Parent+"/") {
				return nil, nil, designError("paint must stay inside its existing parent volume: " + object.ID + "/" + m.ID)
			}
			target[world] = targetCell{object.ID + "/" + m.ID, m.Color}
		}
	}
	visible := map[string]bool{}
	for _, cell := range target {
		visible[cell.Node] = true
	}
	for _, p := range fixed {
		visible[p.Module] = true
	}
	for _, m := range object.Modules {
		prefix := object.ID + "/" + m.ID
		retained := visible[prefix]
		for node := range visible {
			retained = retained || strings.HasPrefix(node, prefix+"/")
		}
		if !retained {
			return nil, nil, designError("paint erased construction " + prefix)
		}
		if m.Design != nil {
			for _, shape := range m.Design.Shapes {
				if shape.Operation == "add" && !visible[prefix+"/"+shape.ID] {
					return nil, nil, designError("paint erased local feature " + prefix + "/" + shape.ID)
				}
			}
		}
	}
	return target, fixed, nil
}

// CompileScene shares the existing tiler, seam repair and authoritative
// validator. One scene has one deadline and search budget across all objects.
// User inventory is advisory; neither planning nor compilation consumes it.
func CompileScene(ctx context.Context, recipe AssemblyRecipe, inventory InventorySnapshot, catalogVersion string, catalog PartCatalog, now time.Time) (CompileResult, error) {
	return CompileSceneWithCache(ctx, recipe, inventory, catalogVersion, catalog, now, NewSceneCompileCache())
}

func CompileSceneWithCache(ctx context.Context, recipe AssemblyRecipe, inventory InventorySnapshot, catalogVersion string, catalog PartCatalog, now time.Time, cache *SceneCompileCache) (CompileResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return CompileResult{Recipe: recipe}, err
	}
	if err := ValidateScene(recipe); err != nil {
		return CompileResult{Recipe: recipe}, err
	}
	resolved, err := resolveSceneAttachments(ctx, *recipe.Scene, catalog)
	if err != nil {
		return CompileResult{Recipe: recipe}, err
	}
	recipe.Scene = &resolved
	if err := ValidateScene(recipe); err != nil {
		return CompileResult{Recipe: recipe}, err
	}
	budget := &searchBudget{ctx: ctx, limit: maxSceneSearchNodes}
	if err := ctx.Err(); err != nil {
		return CompileResult{Recipe: recipe}, err
	}
	parts := []PartSpec{}
	for _, p := range catalog {
		if IsShapePartEligible(p) {
			parts = append(parts, p)
		}
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].ID < parts[j].ID })
	all := []Placement{}
	report := SolverReport{Status: "feasible"}
	for _, object := range recipe.Scene.Objects {
		target, fixed, err := sceneObjectTarget(object, catalog)
		if err != nil {
			return CompileResult{Recipe: recipe}, err
		}
		if err = checkSceneTargetConnected(target, fixed, catalog); err != nil {
			return CompileResult{Recipe: recipe}, fmt.Errorf("object %s: %w", object.ID, err)
		}
		report.TargetCells += len(target)
		hits := 0
		if cache != nil {
			hits = cache.Hits
		}
		found, stages, compileErr := compileSceneObject(ctx, object, target, fixed, parts, catalog, budget, MaxSceneParts-len(all), cache)
		report.CompileStages += stages
		if cache != nil {
			report.ReusedStages += cache.Hits - hits
		}
		if compileErr != nil {
			report.Status = "no_valid_layout"
			report.Visited = budget.used
			if code, _ := BuildErrorCode(compileErr); code == BuildErrorSearchLimit {
				report.Status = "search_limit"
			}
			return CompileResult{Recipe: recipe, Solver: &report}, compileErr
		}
		for i := range found {
			found[i].Object = object.ID
		}
		all = append(all, found...)
		if len(all) > MaxSceneParts {
			return CompileResult{Recipe: recipe}, designError("scene part budget exceeded")
		}
	}
	all = sceneSteps(all, catalog)
	if recipe.Constraints.PartCount > 0 && recipe.Constraints.PartCount != len(all) {
		return CompileResult{Recipe: recipe}, &BuildError{Code: BuildErrorCountUnsupported, Cause: fmt.Errorf("exact count %d does not match compiled %d", recipe.Constraints.PartCount, len(all))}
	}
	if recipe.Constraints.NoWheels {
		for _, p := range all {
			if catalog[p.PartID].Occupancy.GroundContactProfile == "wheel_point" {
				return CompileResult{Recipe: recipe}, designError("no-wheels requirement violated")
			}
		}
	}
	result, err := compilePlacements(recipe, all, UnlimitedInventory(), catalogVersion, catalog, now)
	report.Visited, report.MatchedCells = budget.used, report.TargetCells
	result.Solver = &report
	if err != nil {
		return result, err
	}
	result.Plan.GeneratorVersion = SceneGeneratorVersion
	result.Plan.Validation.Evidence = SceneEvidence(recipe)
	interfaces, interfaceErr := inspectAssemblyInterfaces(result.Plan, *recipe.Scene, catalog)
	if interfaceErr != nil {
		return result, interfaceErr
	}
	result.Plan.Validation.Evidence.Interfaces = interfaces
	if recipe.Scene.Version == 2 {
		estimate := EstimateStaticLoad(ctx, result.Plan, 0.98)
		result.Plan.Validation.Evidence.LoadEstimate = &estimate
	}
	result.Plan.Document = &BuildDocument{Version: 2, Scene: recipe.Scene, DesignHash: SceneHash(*recipe.Scene), Solver: report, ParentCreationID: recipe.Metadata["parent_creation_id"], ParentHash: recipe.Metadata["parent_hash"]}
	result.Plan.InventoryCompared = inventory.Configured
	if inventory.Configured {
		result.Plan.MissingParts = missingSceneParts(all, inventory)
	}
	result.Plan.ContentHash = physicalContentHash(result.Plan)
	result.MPD = ExportMPD(result.Plan)
	return result, nil
}

func SceneHash(scene SceneSpec) string {
	data, _ := json.Marshal(scene)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func sceneSteps(parts []Placement, catalog PartCatalog) []Placement {
	result := append([]Placement{}, parts...)
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		aw, bw := catalog[a.PartID].Occupancy.GroundContactProfile == "wheel_point", catalog[b.PartID].Occupancy.GroundContactProfile == "wheel_point"
		if aw != bw {
			return !aw
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		if a.Object != b.Object {
			return a.Object < b.Object
		}
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		if a.Z != b.Z {
			return a.Z < b.Z
		}
		return a.X < b.X
	})
	step, count := 0, 0
	lastY, lastObject, lastModule := -1, "", ""
	for i := range result {
		p := &result[i]
		if p.Y != lastY || p.Object != lastObject || p.Module != lastModule || count >= 6 {
			step++
			count = 0
			lastY, lastObject, lastModule = p.Y, p.Object, p.Module
		}
		p.Step = step
		count++
	}
	return result
}

func missingSceneParts(parts []Placement, inventory InventorySnapshot) []InventoryItem {
	counts := map[inventoryKey]int{}
	for _, p := range parts {
		counts[inventoryKey{p.PartID, p.Color}]++
	}
	available := inventory.quantities()
	missing := []InventoryItem{}
	for key, n := range counts {
		if n > available[key] {
			missing = append(missing, InventoryItem{key.partID, key.color, n - available[key]})
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].PartID != missing[j].PartID {
			return missing[i].PartID < missing[j].PartID
		}
		return missing[i].Color < missing[j].Color
	})
	return missing
}

func validateScenePlacements(parts []Placement, scene *SceneSpec, catalog PartCatalog) ValidationReport {
	report := ValidationReport{Issues: []ValidationIssue{}, UsedParts: map[string]int{}, PartCount: len(parts)}
	objects := map[string]bool{}
	for _, o := range scene.Objects {
		objects[o.ID] = true
	}
	occupied := map[DesignVector]Placement{}
	ids := map[string]bool{}
	for _, p := range parts {
		if !objects[p.Object] || ids[p.ID] {
			report.Issues = append(report.Issues, ValidationIssue{Code: "invalid_object", Message: "零件对象或标识无效", PlacementID: p.ID})
		}
		ids[p.ID] = true
		report.UsedParts[p.PartID]++
		report.StepCount = max(report.StepCount, p.Step)
		sz := orientedSizeWith(p, catalog)
		for x := 0; x < sz.x; x++ {
			for z := 0; z < sz.z; z++ {
				for y := 0; y < catalog[p.PartID].PlatesY; y++ {
					q := DesignVector{p.X + x, p.Y + y, p.Z + z}
					if prev, ok := occupied[q]; ok && prev.Object != p.Object {
						report.Issues = append(report.Issues, ValidationIssue{Code: "collision", Message: "场景对象发生重叠", PlacementID: p.ID})
					}
					occupied[q] = p
				}
			}
		}
	}
	first := true
	for _, o := range scene.Objects {
		group := []Placement{}
		for _, p := range parts {
			if p.Object == o.ID {
				group = append(group, p)
			}
		}
		r := validateAssembly(group, deriveExactConnections(group, catalog), UnlimitedInventory(), catalog, -64, 64)
		report.Issues = append(report.Issues, r.Issues...)
		report.ConnectionCount += r.ConnectionCount
		if first || r.MinimumStabilityMarginMils < report.MinimumStabilityMarginMils {
			report.MinimumStabilityMarginMils = r.MinimumStabilityMarginMils
			first = false
		}
	}
	report.Buildable = len(report.Issues) == 0
	return report
}

func SceneEvidence(recipe AssemblyRecipe) *BuildEvidence {
	e := &BuildEvidence{Objects: len(recipe.Scene.Objects), Geometry: "catalog_occupancy_and_exact_connectors", Steps: "earlier_step_support_and_static_volume_mass", Semantic: "requirements_attributed_not_proven", Physical: "not_tested", RetailColors: "not_verified", Requirements: []RequirementEvidence{}}
	for n, r := range recipe.Requirements {
		entry := RequirementEvidence{Requirement: r, Modules: []string{}}
		for _, o := range recipe.Scene.Objects {
			for _, m := range o.Modules {
				for _, index := range m.Requirements {
					if index == n {
						entry.Modules = append(entry.Modules, o.ID+"/"+m.ID)
					}
				}
			}
		}
		e.Requirements = append(e.Requirements, entry)
	}
	for _, o := range recipe.Scene.Objects {
		e.Modules += len(o.Modules)
	}
	return e
}
