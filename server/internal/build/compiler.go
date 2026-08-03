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

func Compile(recipe AssemblyRecipe, now time.Time) (CompileResult, error) {
	placements := placementsFor(recipe)
	report := Validate(placements)
	plan := BuildPlan{
		Version: 1, KitID: StarterKitID, CatalogVersion: CatalogVersion,
		ModuleLibraryVersion: ModuleLibraryVersion, CompilerVersion: CompilerVersion,
		ValidatorVersion: ValidatorVersion, Title: recipe.Title, Prompt: recipe.Prompt,
		Archetype: recipe.Archetype, Placements: placements, Connections: deriveConnections(placements),
		Steps: deriveSteps(placements, report.StepCount), Parts: StarterCatalog,
		Validation: report, GeneratedAt: now.UTC(),
	}
	plan.ContentHash = physicalContentHash(plan)
	if !report.Buildable {
		return CompileResult{Recipe: recipe, Plan: plan}, fmt.Errorf("compiled plan is not buildable: %v", report.Issues)
	}
	return CompileResult{Recipe: recipe, Plan: plan, MPD: ExportMPD(plan)}, nil
}

func deriveConnections(placements []Placement) []Connection {
	connections := make([]Connection, 0)
	for i := 0; i < len(placements); i++ {
		for j := i + 1; j < len(placements); j++ {
			if !placementsConnect(placements[i], placements[j]) {
				continue
			}
			kind := "stud"
			if placements[i].Module == "wheels" || placements[j].Module == "wheels" {
				kind = "wheel_pin"
			}
			connections = append(connections, Connection{
				ID: fmt.Sprintf("c%02d", len(connections)+1), APlacementID: placements[i].ID,
				BPlacementID: placements[j].ID, Kind: kind,
			})
		}
	}
	return connections
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
		KitID                string      `json:"kit_id"`
		CatalogVersion       string      `json:"catalog_version"`
		ModuleLibraryVersion string      `json:"module_library_version"`
		CompilerVersion      string      `json:"compiler_version"`
		ValidatorVersion     string      `json:"validator_version"`
		Archetype            string      `json:"archetype"`
		Placements           []Placement `json:"placements"`
	}{plan.KitID, plan.CatalogVersion, plan.ModuleLibraryVersion, plan.CompilerVersion, plan.ValidatorVersion, plan.Archetype, plan.Placements}
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
		// rim+tyre shortcut. Two holders are bridged by a rotated 2x4 plate.
		for i, z := range []int{0, 2} {
			step := 1 + i
			add("wheel-holder-2x2", 71, 0, 0, z, 0, step, "rolling-base")
			add("wheel", 71, -1, 0, z, 90, step, "wheels")
			add("wheel", 71, 2, 0, z, 270, step, "wheels")
		}
		add("plate-2x4", 4, 0, 1, 0, 90, 3, "rolling-base")
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
		add("brick-2x2", 1, 0, 0, 0, 0, 1, "left-foot")
		add("brick-2x2", 1, 3, 0, 0, 0, 1, "right-foot")
		add("brick-2x4", 4, 0, 3, 0, 0, 2, "torso")
		add("brick-2x2", 14, 1, 6, 0, 0, 3, "head")
		add("brick-1x2", 15, 0, 6, 0, 90, 4, "left-arm")
		add("brick-1x2", 15, 3, 6, 0, 90, 4, "right-arm")
		add("brick-1x1", 1, 1, 9, 0, 0, 5, "antenna")
	default:
		add("brick-2x4", 2, 0, 0, 0, 0, 1, "body")
		add("brick-2x2", 14, 0, 3, 0, 0, 2, "head")
		add("brick-1x1", 15, 0, 6, 0, 0, 3, "left-eye")
		add("brick-1x1", 15, 1, 6, 0, 0, 3, "right-eye")
		add("plate-1x2", 4, 3, 3, 0, 0, 4, "tail")
		add("slope-2x2", 4, 4, 4, 0, 0, 5, "tail-tip")
	}
	return p
}

func Validate(placements []Placement) ValidationReport {
	report := ValidationReport{Buildable: true, UsedParts: map[string]int{}, PartCount: len(placements)}
	maxStep := 0
	occupied := map[[3]int]string{}
	for _, p := range placements {
		spec, ok := StarterCatalog[p.PartID]
		if !ok {
			report.Issues = append(report.Issues, ValidationIssue{Code: "unknown_part", Message: "零件不在套装目录中", PlacementID: p.ID})
			continue
		}
		if !allowedColors[p.Color] {
			report.Issues = append(report.Issues, ValidationIssue{Code: "unknown_color", Message: "颜色不在套装目录中", PlacementID: p.ID})
		}
		report.UsedParts[p.PartID]++
		if report.UsedParts[p.PartID] > spec.Quantity {
			report.Issues = append(report.Issues, ValidationIssue{Code: "inventory_exceeded", Message: "使用数量超过套装库存", PlacementID: p.ID})
		}
		if p.Y < 0 || p.X < -16 || p.X > 16 || p.Z < -16 || p.Z > 16 {
			report.Issues = append(report.Issues, ValidationIssue{Code: "out_of_bounds", Message: "零件超出安全搭建范围", PlacementID: p.ID})
		}
		if p.Rotation%90 != 0 {
			report.Issues = append(report.Issues, ValidationIssue{Code: "invalid_rotation", Message: "零件角度必须按 90° 旋转", PlacementID: p.ID})
		}
		if p.Step > maxStep {
			maxStep = p.Step
		}
		sx, sz := spec.StudsX, spec.StudsZ
		if p.Rotation%180 != 0 {
			sx, sz = sz, sx
		}
		for x := p.X; x < p.X+sx; x++ {
			for z := p.Z; z < p.Z+sz; z++ {
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

	// Every elevated part must have a legal stud surface beneath it from an
	// earlier (or same) step. Wheel modules are the one starter-kit exception:
	// they connect laterally through their axle module.
	for _, placement := range placements {
		if placement.Y == 0 || placement.Module == "wheels" {
			continue
		}
		supported := false
		for _, candidate := range placements {
			if candidate.ID == placement.ID || candidate.Step >= placement.Step {
				continue
			}
			if topY(candidate) == placement.Y && horizontalOverlap(placement, candidate) {
				supported = true
				break
			}
		}
		if !supported {
			report.Issues = append(report.Issues, ValidationIssue{Code: "unsupported", Message: "零件下方没有可连接的凸点", PlacementID: placement.ID})
		}
	}

	// Connectivity is evaluated as one assembly graph. Vertical stud contact
	// forms ordinary edges; wheels may form a lateral axle edge.
	if len(placements) > 0 {
		visited := map[string]bool{placements[0].ID: true}
		changed := true
		for changed {
			changed = false
			for _, candidate := range placements {
				if visited[candidate.ID] {
					continue
				}
				for _, connected := range placements {
					if visited[connected.ID] && placementsConnect(candidate, connected) {
						visited[candidate.ID] = true
						changed = true
						break
					}
				}
			}
		}
		for _, placement := range placements {
			if !visited[placement.ID] {
				report.Issues = append(report.Issues, ValidationIssue{Code: "disconnected", Message: "零件没有连接到主体", PlacementID: placement.ID})
			}
		}
	}

	// A conservative stability check runs for every cumulative build step, not
	// just the attractive final pose. This prevents instructions that pass only
	// after a later counterweight has been added.
	for step := 1; step <= maxStep; step++ {
		cumulative := make([]Placement, 0, len(placements))
		for _, placement := range placements {
			if placement.Step <= step {
				cumulative = append(cumulative, placement)
			}
		}
		if !centerOverBase(cumulative) {
			report.Issues = append(report.Issues, ValidationIssue{Code: "unstable_step", Message: fmt.Sprintf("第 %d 步的重心超出了底座范围", step)})
		}
	}
	report.StepCount = maxStep
	report.Buildable = len(report.Issues) == 0
	sort.Slice(report.Issues, func(i, j int) bool { return report.Issues[i].Code < report.Issues[j].Code })
	return report
}

func orientedSize(placement Placement) (x, z int) {
	spec := StarterCatalog[placement.PartID]
	x, z = spec.StudsX, spec.StudsZ
	if placement.Rotation%180 != 0 {
		x, z = z, x
	}
	return x, z
}

func topY(placement Placement) int { return placement.Y + StarterCatalog[placement.PartID].PlatesY }

func horizontalOverlap(a, b Placement) bool {
	ax, az := orientedSize(a)
	bx, bz := orientedSize(b)
	return a.X < b.X+bx && b.X < a.X+ax && a.Z < b.Z+bz && b.Z < a.Z+az
}

func placementsConnect(a, b Placement) bool {
	if horizontalOverlap(a, b) && (topY(a) == b.Y || topY(b) == a.Y) {
		return true
	}
	if a.Module != "wheels" && b.Module != "wheels" {
		return false
	}
	ax, az := orientedSize(a)
	bx, bz := orientedSize(b)
	xGap := max(a.X, b.X) - min(a.X+ax, b.X+bx)
	zGap := max(a.Z, b.Z) - min(a.Z+az, b.Z+bz)
	yOverlap := a.Y < topY(b) && b.Y < topY(a)
	return yOverlap && xGap <= 1 && zGap <= 1
}

func centerOverBase(placements []Placement) bool {
	if len(placements) == 0 {
		return false
	}
	minX, minZ, maxX, maxZ := 1<<30, 1<<30, -1<<30, -1<<30
	weightedX, weightedZ, volume := 0.0, 0.0, 0.0
	for _, placement := range placements {
		spec, ok := StarterCatalog[placement.PartID]
		if !ok {
			continue
		}
		sx, sz := orientedSize(placement)
		partVolume := float64(sx * sz * spec.PlatesY)
		weightedX += (float64(placement.X) + float64(sx)/2) * partVolume
		weightedZ += (float64(placement.Z) + float64(sz)/2) * partVolume
		volume += partVolume
		if placement.Y == 0 {
			minX, minZ = min(minX, placement.X), min(minZ, placement.Z)
			maxX, maxZ = max(maxX, placement.X+sx), max(maxZ, placement.Z+sz)
		}
	}
	if volume == 0 || minX > maxX {
		return false
	}
	centerX, centerZ := weightedX/volume, weightedZ/volume
	return centerX >= float64(minX) && centerX <= float64(maxX) && centerZ >= float64(minZ) && centerZ <= float64(maxZ)
}
