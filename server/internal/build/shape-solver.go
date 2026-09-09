package build

import (
	"context"
	"fmt"
	"sort"
	"time"
)

const maxShapeSearchNodes = 24000

// One invocation owns one budget, including every restart and local repair.
// Searches are sequential; local quotas measure shared usage since entry so
// repairs retain their existing cost against the enclosing attempt's quota.
type searchBudget struct {
	ctx            context.Context
	limit, used    int
	repairNodes    int
	repairAttempts int
	repairs        int
	pruned         int
}

func (b *searchBudget) exhausted() bool {
	return b.used >= b.limit || b.ctx.Err() != nil
}

func (b *searchBudget) consume(repair bool) bool {
	if b.exhausted() {
		return false
	}
	b.used++
	if repair {
		b.repairNodes++
	}
	return true
}

// IsShapePartEligible identifies reviewed solid parts usable for target tiling.
// Placement-specific geometry, stock and support still require validation.
func IsShapePartEligible(part PartSpec) bool {
	return partIsMechanicallyCertified(part) && part.GeometryProfile == "stud_tube_rect" && part.HasTopStuds && part.HasBottomReceptors
}

type shapeCandidate struct {
	Placement Placement
	Cells     []DesignVector
	Score     int
}
type shapeSearch struct {
	target       designTarget
	cells        []DesignVector
	parts        []PartSpec
	catalog      PartCatalog
	inventory    InventorySnapshot
	remaining    map[inventoryKey]int
	occupied     map[DesignVector]int
	placements   []Placement
	exactColors  bool
	exactCount   int
	start        int
	limit        int
	budget       *searchBudget
	variant      int
	limited      bool
	triedRepair  bool
	seamPriority bool
	accept       func([]Placement) bool
	result       []Placement
}

// SolveDesign searches a finite catalog-backed tiling space. It may report a
// search limit, never "impossible" merely because its budget was exhausted.
func SolveDesign(ctx context.Context, recipe AssemblyRecipe, inventory InventorySnapshot, catalog PartCatalog) (placements []Placement, report SolverReport, err error) {
	report = SolverReport{Status: "invalid", StopReason: "invalid_input"}
	budget := &searchBudget{ctx: ctx, limit: maxShapeSearchNodes}
	defer func() {
		report.Visited, report.RepairNodes = budget.used, budget.repairNodes
		report.RepairAttempts, report.Repairs = budget.repairAttempts, budget.repairs
		report.Pruned = budget.pruned
	}()
	if recipe.Design == nil || len(recipe.Modules) != 0 || recipe.Version != 3 || len(recipe.Constraints.RequiredModules) != 0 {
		return nil, report, designError("shape recipes require version 3 and no module constraints")
	}
	if recipe.Constraints.PartCount < 0 || recipe.Constraints.PartCount > 200 {
		return nil, report, &BuildError{Code: BuildErrorCountUnsupported, Cause: fmt.Errorf("part count outside supported range")}
	}
	target, err := RasterizeDesign(*recipe.Design)
	if err != nil {
		return nil, report, err
	}
	report.TargetCells = len(target)
	parts := []PartSpec{}
	for _, part := range catalog {
		// Only reviewed solid stud/tube parts can tile arbitrary volumes. Special
		// geometry retains its separate certified module path.
		if IsShapePartEligible(part) {
			parts = append(parts, part)
		}
	}
	sort.Slice(parts, func(i, j int) bool { return parts[i].ID < parts[j].ID })
	if len(parts) == 0 {
		report.StopReason = "no_eligible_parts"
		return nil, report, &BuildError{Code: BuildErrorUnsupported, Cause: fmt.Errorf("no reviewed solid parts in catalog")}
	}
	if inventory.Configured {
		available := 0
		colors := map[int]int{}
		for _, item := range inventory.Items {
			p, ok := catalog[item.PartID]
			if ok && IsShapePartEligible(p) {
				v := p.StudsX * p.StudsZ * p.PlatesY * item.Quantity
				available += v
				colors[item.Color] += v
			}
		}
		needed := map[int]int{}
		for _, cell := range target {
			needed[cell.Color]++
		}
		missing := available < len(target)
		if recipe.Constraints.ExactColors {
			for c, n := range needed {
				if colors[c] < n {
					missing = true
				}
			}
		}
		if missing {
			report.Status = "insufficient_inventory"
			report.StopReason = "inventory_insufficient"
			return nil, report, &BuildError{Code: BuildErrorInsufficientInventory, Cause: fmt.Errorf("available reusable parts cannot cover this model's target volume")}
		}
	}
	limited := false
	// Stock-constrained repairs need neighborhood exchanges when pair swaps
	// cannot use extra parts. Keep all six orientation seeds and the original
	// ordering for unrestricted catalogs; neither heuristic fits every shape.
	for variant := 0; variant < 6; variant++ {
		remainingBudget := budget.limit - budget.used
		if remainingBudget <= 0 {
			limited = true
			break
		}
		report.Attempts = variant + 1
		search := &shapeSearch{target: target, cells: sortedTargetCells(target), parts: parts, catalog: catalog, inventory: inventory,
			remaining: inventory.quantities(), occupied: map[DesignVector]int{}, exactColors: recipe.Constraints.ExactColors,
			exactCount: recipe.Constraints.PartCount, start: budget.used, limit: min(maxShapeSearchNodes/6, remainingBudget), budget: budget, variant: variant, seamPriority: inventory.Configured && variant > 0}
		if search.walk(0) {
			report.Status, report.MatchedCells = "feasible", len(target)
			report.StopReason = "feasible"
			return normalizeShapeSteps(search.result), report, nil
		}
		limited = limited || search.limited
		if ctx.Err() != nil {
			limited = true
			break
		}
		if !search.limited {
			break
		}
	}
	if limited {
		report.Status = "search_limit"
		report.StopReason = "node_limit"
		if ctx.Err() == context.DeadlineExceeded {
			report.StopReason = "deadline"
		} else if ctx.Err() != nil {
			report.StopReason = "cancelled"
		}
		return nil, report, &BuildError{Code: BuildErrorSearchLimit, Cause: fmt.Errorf("shape search stopped (%s) after %d nodes", report.StopReason, budget.used)}
	}
	report.Status = "no_valid_layout"
	report.StopReason = "no_valid_layout"
	if budget.used == 1 {
		report.StopReason = "no_candidates"
		if budget.pruned > 0 {
			report.StopReason = "capacity_bound"
		}
	}
	return nil, report, &BuildError{Code: BuildErrorStructureInvalid, Cause: fmt.Errorf("no validated layout found for the target and current parts")}
}

func (s *shapeSearch) walk(cursor int) bool {
	if s.budget.used-s.start >= s.limit || !s.budget.consume(s.accept != nil) {
		s.limited = true
		return false
	}
	for cursor < len(s.cells) {
		if _, ok := s.occupied[s.cells[cursor]]; !ok {
			break
		}
		cursor++
	}
	if cursor == len(s.cells) {
		if s.exactCount > 0 && len(s.placements) != s.exactCount {
			return false
		}
		placements := normalizeShapeSteps(s.placements)
		if s.accept != nil {
			if s.accept(placements) {
				s.result = placements
				return true
			}
			return false
		}
		validation := ValidateWithCatalog(placements, s.inventory, s.catalog)
		if validation.Buildable {
			s.result = placements
			return true
		}
		if !s.triedRepair && onlyDisconnectedIssues(validation) {
			s.triedRepair = true
			if repaired, ok := s.repairSeams(placements); ok {
				s.result = repaired
				return true
			}
		}
		return false
	}
	if len(s.placements) >= 200 || (s.exactCount > 0 && len(s.placements) >= s.exactCount) {
		return false
	}
	if !s.canCoverRemaining() {
		s.budget.pruned++
		return false
	}
	candidates := s.candidates(s.cells[cursor])
	for _, candidate := range candidates {
		p := candidate.Placement
		p.ID = fmt.Sprintf("%s:%d:%d:%d", p.Module, p.X, p.Y, p.Z)
		index := len(s.placements)
		s.placements = append(s.placements, p)
		for _, cell := range candidate.Cells {
			s.occupied[cell] = index
		}
		key := inventoryKey{p.PartID, p.Color}
		s.remaining[key]--
		if s.walk(cursor + 1) {
			return true
		}
		s.remaining[key]++
		for _, cell := range candidate.Cells {
			delete(s.occupied, cell)
		}
		s.placements = s.placements[:index]
		if s.limited {
			return false
		}
	}
	return false
}

func (s *shapeSearch) candidates(cell DesignVector) []shapeCandidate {
	value := s.target[cell]
	result := []shapeCandidate{}
	parents := make([]int, len(s.placements))
	for i := range parents {
		parents[i] = i
	}
	var root func(int) int
	root = func(i int) int {
		for parents[i] != i {
			i = parents[i]
		}
		return i
	}
	for i, p := range s.placements {
		size := orientedSizeWith(p, s.catalog)
		for x := 0; x < size.x; x++ {
			for z := 0; z < size.z; z++ {
				if below, ok := s.occupied[DesignVector{p.X + x, p.Y - 1, p.Z + z}]; ok && below != i {
					parents[root(i)] = root(below)
				}
			}
		}
	}
	for _, part := range s.parts {
		if s.budget.ctx.Err() != nil {
			s.limited = true
			return nil
		}
		for rotation := 0; rotation <= 90; rotation += 90 {
			if rotation == 90 && part.StudsX == part.StudsZ {
				continue
			}
			sx, sz := part.StudsX, part.StudsZ
			if rotation == 90 {
				sx, sz = sz, sx
			}
			// The first uncovered cell is minimal in Y/Z/X. An eligible new solid
			// cannot start before it without covering an absent or occupied cell.
			p := Placement{PartID: part.ID, Color: value.Color, X: cell.X, Y: cell.Y, Z: cell.Z, Rotation: rotation, Step: cell.Y + 1, Module: value.Node}
			volume := sx * sz * part.PlatesY
			cells := make([]DesignVector, 0, volume)
			valid := true
			for y := 0; y < part.PlatesY && valid; y++ {
				for z := 0; z < sz && valid; z++ {
					for x := 0; x < sx; x++ {
						q := DesignVector{cell.X + x, cell.Y + y, cell.Z + z}
						v, ok := s.target[q]
						_, occupied := s.occupied[q]
						if !ok || occupied || v != value {
							valid = false
							break
						}
						cells = append(cells, q)
					}
				}
			}
			if !valid {
				continue
			}
			seams := 0
			if s.seamPriority {
				seams = s.seamDepth(cell, sx, sz)
			}
			supports := map[int]bool{}
			components := map[int]bool{}
			engaged := 0
			if cell.Y > 0 {
				for x := 0; x < sx; x++ {
					for z := 0; z < sz; z++ {
						if index, ok := s.occupied[DesignVector{cell.X + x, cell.Y - 1, cell.Z + z}]; ok {
							below := s.placements[index]
							if below.Y+s.catalog[below.PartID].PlatesY == cell.Y {
								engaged++
								supports[index] = true
								components[root(index)] = true
							}
						}
					}
				}
				if engaged < minimumSupportEngagement(p, s.catalog) {
					continue
				}
			}
			if s.seamPriority {
				// During local retiling, frozen parts may also exist above the
				// candidate. BrickGPT's component priority considers both faces.
				// Upper contacts do not count as support from an earlier step.
				for x := 0; x < sx; x++ {
					for z := 0; z < sz; z++ {
						if index, ok := s.occupied[DesignVector{cell.X + x, cell.Y + part.PlatesY, cell.Z + z}]; ok && s.placements[index].Y == cell.Y+part.PlatesY {
							components[root(index)] = true
						}
					}
				}
			}
			colors := []int{value.Color}
			if s.inventory.Configured && !s.exactColors {
				for _, c := range AllowedColorCodes() {
					if c != value.Color {
						colors = append(colors, c)
					}
				}
			}
			for _, color := range colors {
				if s.inventory.Configured && s.remaining[inventoryKey{part.ID, color}] <= 0 {
					continue
				}
				p.Color = color
				score := shapeCandidateScore(p, part, sx, sz, len(supports), len(components), seams, value.Color, s.variant)
				result = append(result, shapeCandidate{p, cells, score})
			}
		}
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].Score > result[j].Score })
	return result
}

// These optimistic capacity bounds cannot reject a feasible completion.
// Disconnected partial layouts may still acquire a cross-brick later.
func (s *shapeSearch) canCoverRemaining() bool {
	uncovered := len(s.target) - len(s.occupied)
	availableVolume, availableCount, largest := 0, 0, 0
	for _, part := range s.parts {
		volume := part.StudsX * part.StudsZ * part.PlatesY
		if !s.inventory.Configured {
			largest = max(largest, volume)
			continue
		}
		for _, color := range AllowedColorCodes() {
			quantity := s.remaining[inventoryKey{part.ID, color}]
			if quantity <= 0 {
				continue
			}
			largest = max(largest, volume)
			availableVolume += quantity * volume
			availableCount += quantity
		}
	}
	if largest == 0 || (s.inventory.Configured && availableVolume < uncovered) {
		return false
	}
	countLimit := 200
	if s.exactCount > 0 {
		countLimit = s.exactCount
		if s.inventory.Configured && availableCount < s.exactCount-len(s.placements) {
			return false
		}
	}
	return len(s.placements)+(uncovered+largest-1)/largest <= countLimit
}

// Scores order eligible candidates; they never override a hard constraint.
func shapeCandidateScore(p Placement, part PartSpec, sx, sz, supports, components, seams, targetColor, variant int) int {
	const volumeWeight, supportWeight, componentWeight, seamWeight = 12, 45, 400, 12
	const colorWeight, orientationWeight = 1000, 8
	score := sx*sz*part.PlatesY*volumeWeight + supports*supportWeight + max(0, components-1)*componentWeight + seams*seamWeight
	if p.Color == targetColor {
		score += colorWeight
	}
	if (p.Y/3+variant)%2 == p.Rotation/90 {
		score += orientationWeight
	}
	// Deterministic restarts explore different seam patterns.
	if variant > 0 {
		score += ((sx*31 + sz*17 + part.PlatesY*13 + p.X*7 + p.Z*11 + variant*19) * (variant + 3) % 43) * orientationWeight
	}
	return score
}

func normalizeShapeSteps(placements []Placement) []Placement {
	result := append([]Placement{}, placements...)
	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		if a.Z != b.Z {
			return a.Z < b.Z
		}
		return a.X < b.X
	})
	levels := []int{}
	seen := map[int]bool{}
	for _, p := range result {
		if !seen[p.Y] {
			levels = append(levels, p.Y)
			seen[p.Y] = true
		}
	}
	sort.Ints(levels)
	steps := map[int]int{}
	for i, y := range levels {
		steps[y] = i + 1
	}
	for i := range result {
		result[i].Step = steps[result[i].Y]
	}
	return result
}

// CompileDesign uses the same final validator and exchange format as modules.
func CompileDesign(ctx context.Context, recipe AssemblyRecipe, inventory InventorySnapshot, catalogVersion string, catalog PartCatalog, now time.Time) (CompileResult, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	placements, solver, err := SolveDesign(ctx, recipe, inventory, catalog)
	if err != nil {
		return CompileResult{Recipe: recipe, Solver: &solver}, err
	}
	result, err := compilePlacements(recipe, placements, inventory, catalogVersion, catalog, now)
	result.Solver = &solver
	if err != nil {
		return result, err
	}
	result.Plan.GeneratorVersion = ShapeGeneratorVersion
	result.Plan.Document = &BuildDocument{Version: 1, Design: *recipe.Design, DesignHash: DesignHash(*recipe.Design), Solver: solver,
		ParentCreationID: recipe.Metadata["parent_creation_id"], ParentHash: recipe.Metadata["parent_hash"]}
	result.Plan.ContentHash = physicalContentHash(result.Plan)
	result.MPD = ExportMPD(result.Plan)
	return result, nil
}
