package build

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCompileStarterArchetypesAreBuildableAndDeterministic(t *testing.T) {
	now := time.Date(2026, 8, 3, 8, 0, 0, 0, time.UTC)
	for _, prompt := range []string{"会跑的红色小车", "有翅膀的蓝色小龙", "勇敢的机器人", "会摇尾巴的小狗"} {
		recipe := PlanRecipe(prompt, nil)
		first, err := Compile(recipe, UnlimitedInventory(), now)
		if err != nil {
			t.Fatalf("Compile(%q): %v", prompt, err)
		}
		second, err := Compile(recipe, UnlimitedInventory(), now)
		if err != nil {
			t.Fatalf("second Compile(%q): %v", prompt, err)
		}
		if !first.Plan.Validation.Buildable {
			t.Fatalf("%q plan not buildable: %#v", prompt, first.Plan.Validation.Issues)
		}
		if first.Plan.Version != 3 || first.Plan.ConnectorSchemaVersion != ConnectorSchemaVersion || first.Plan.PhysicsProfileVersion == "" || first.Plan.GeneratorVersion == "" {
			t.Fatalf("%q plan lacks versioned mechanics metadata: %#v", prompt, first.Plan)
		}
		if first.Plan.Validation.ConnectionCount != len(first.Plan.Connections) || len(first.Plan.Connections) == 0 {
			t.Fatalf("%q plan lacks exact connection evidence: %#v", prompt, first.Plan.Connections)
		}
		for _, connection := range first.Plan.Connections {
			if connection.EngagedCount <= 0 || len(connection.AConnectorIDs) != connection.EngagedCount || len(connection.BConnectorIDs) != connection.EngagedCount {
				t.Fatalf("%q invalid connection evidence: %#v", prompt, connection)
			}
		}
		wire, err := json.Marshal(first.Plan)
		if err != nil {
			t.Fatalf("marshal %q plan: %v", prompt, err)
		}
		for _, required := range []string{`"issues":[]`} {
			if !strings.Contains(string(wire), required) {
				t.Fatalf("%q plan JSON lacks stable empty array %s: %s", prompt, required, wire)
			}
		}
		if strings.Contains(string(wire), `"inventory":`) {
			t.Fatal("new creations must not freeze reusable inventory")
		}
		if first.MPD != second.MPD {
			t.Fatalf("%q export is not deterministic", prompt)
		}
		if !strings.Contains(first.MPD, "3001.dat") && !strings.Contains(first.MPD, "3003.dat") {
			t.Fatalf("%q MPD lacks official part ids", prompt)
		}
	}
}

func TestValidateRejectsCollisionAndUnknownPart(t *testing.T) {
	report := Validate([]Placement{
		{ID: "one", PartID: "brick-2x2", Color: 4},
		{ID: "two", PartID: "brick-2x2", Color: 4},
		{ID: "three", PartID: "imaginary", Color: 4},
	}, UnlimitedInventory())
	if report.Buildable {
		t.Fatal("expected invalid plan")
	}
	codes := map[string]bool{}
	for _, issue := range report.Issues {
		codes[issue.Code] = true
	}
	if !codes["collision"] || !codes["unknown_part"] {
		t.Fatalf("missing issues: %#v", report.Issues)
	}
}

func TestRacerUsesOfficialCompatibleWheelHolderAndCenteredLDrawCoordinates(t *testing.T) {
	result, err := Compile(PlanRecipe("一辆会跑的红色小车", nil), UnlimitedInventory(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"4600.dat", "4624c04.dat", "1 4 20 -16 40", "1 71 -10 -3 20"} {
		if !strings.Contains(result.MPD, required) {
			t.Fatalf("MPD missing %q:\n%s", required, result.MPD)
		}
	}
	if strings.Contains(strings.ReplaceAll(result.MPD, "\r\n", ""), "\n") {
		t.Fatal("MPD must use canonical CRLF line endings")
	}
}

func TestPhysicalContentHashIgnoresPresentationAndGenerationTime(t *testing.T) {
	firstRecipe := PlanRecipe("会飞的小龙", nil)
	secondRecipe := firstRecipe
	secondRecipe.Title = "另一个名字"
	secondRecipe.Prompt = "同一结构的另一个说法"
	first, err := Compile(firstRecipe, UnlimitedInventory(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Compile(secondRecipe, UnlimitedInventory(), time.Unix(999, 0))
	if err != nil {
		t.Fatal(err)
	}
	if first.Plan.ContentHash == "" || first.Plan.ContentHash != second.Plan.ContentHash {
		t.Fatalf("physical hash drifted: %q != %q", first.Plan.ContentHash, second.Plan.ContentHash)
	}
	if len(first.Plan.Connections) == 0 || len(first.Plan.Steps) != first.Plan.Validation.StepCount {
		t.Fatalf("plan graph incomplete: %#v", first.Plan)
	}
}

func TestConfiguredInventoryRestrictsArchetypesAndReassignsColors(t *testing.T) {
	unlimited, err := Compile(PlanRecipe("会摇尾巴的小狗", nil), UnlimitedInventory(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	for _, placement := range unlimited.Plan.Placements {
		counts[placement.PartID]++
	}
	items := make([]InventoryItem, 0, len(counts))
	for partID, quantity := range counts {
		items = append(items, InventoryItem{PartID: partID, Color: 1, Quantity: quantity})
	}
	inventory := NewInventorySnapshot(true, 1, items)
	available := AvailableArchetypes(inventory)
	if len(available) != 1 || available[0] != "creature" {
		t.Fatalf("available archetypes = %#v, want creature", available)
	}
	result, err := Compile(PlanRecipe("会摇尾巴的小狗", nil), inventory, time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, placement := range result.Plan.Placements {
		if placement.Color != 1 {
			t.Fatalf("placement %s color = %d, want owned blue", placement.ID, placement.Color)
		}
	}
	nextRevision, err := Compile(PlanRecipe("会摇尾巴的小狗", nil), NewInventorySnapshot(true, 2, items), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if result.Plan.ContentHash != nextRevision.Plan.ContentHash {
		t.Fatal("inventory audit revision changed the physical content hash")
	}
}

func TestEmptyConfiguredInventoryCannotBuildButUnlimitedCan(t *testing.T) {
	recipe := PlanRecipe("勇敢的机器人", nil)
	if _, err := Compile(recipe, UnlimitedInventory(), time.Unix(0, 0)); err != nil {
		t.Fatalf("unlimited inventory should compile: %v", err)
	}
	configured := NewInventorySnapshot(true, 1, nil)
	result, err := Compile(recipe, configured, time.Unix(0, 0))
	if err == nil || result.Plan.Validation.Buildable {
		t.Fatal("empty configured inventory should reject the plan")
	}
	if available := AvailableArchetypes(configured); len(available) != 0 {
		t.Fatalf("empty inventory unexpectedly supports %#v", available)
	}
}

func TestDifficultyPolicyBuildsEveryArchetypeAtEveryLevel(t *testing.T) {
	for _, archetype := range []string{"racer", "flyer", "robot", "creature"} {
		for level := 1; level <= 5; level++ {
			recipe := PlanRecipe("难度 "+strconv.Itoa(level), nil)
			recipe = ExampleRecipe(archetype)
			recipe = ApplyDifficultyPolicy(recipe, recipe.Prompt, nil)
			result, err := Compile(recipe, UnlimitedInventory(), time.Unix(0, 0))
			if err != nil {
				t.Fatalf("%s level %d: %v", archetype, level, err)
			}
			want := len(placementsFor(recipe))
			if result.Plan.Validation.PartCount != want {
				t.Fatalf("%s level %d count = %d, want reviewed module count %d", archetype, level, result.Plan.Validation.PartCount, want)
			}
			for _, placement := range result.Plan.Placements {
				if placement.Module == "difficulty-detail" {
					t.Fatalf("%s level %d used unsafe filler placement %#v", archetype, level, placement)
				}
			}
		}
	}
}

func TestExplicitPartCountWinsAndInventoryFailsWithoutRetryableError(t *testing.T) {
	recipe := ApplyDifficultyPolicy(PlanRecipe("6岁，难度5，用37块积木做机器人", nil), "6岁，难度5，用37块积木做机器人", nil)
	if targetPartCount(recipe) != 37 || recipe.Metadata["part_count_source"] != "explicit" {
		t.Fatalf("unexpected policy: %#v", recipe.Metadata)
	}
	_, err := Compile(recipe, UnlimitedInventory(), time.Unix(0, 0))
	if code, ok := BuildErrorCode(err); !ok || code != BuildErrorCountUnsupported {
		t.Fatalf("unsupported exact count error = %v, code = %q", err, code)
	}

	exact := ApplyDifficultyPolicy(PlanRecipe("用7块积木做机器人", nil), "用7块积木做机器人", nil)
	result, err := Compile(exact, UnlimitedInventory(), time.Unix(0, 0))
	if err != nil || result.Plan.Validation.PartCount != 7 {
		t.Fatalf("reviewed exact count compile = %d, %v", result.Plan.Validation.PartCount, err)
	}
	_, err = Compile(exact, NewInventorySnapshot(true, 1, nil), time.Unix(0, 0))
	if code, ok := BuildErrorCode(err); !ok || code != BuildErrorInsufficientInventory {
		t.Fatalf("inventory error = %v, code = %q", err, code)
	}
}

func TestExactConnectorsRejectBoundingBoxPhantomConnection(t *testing.T) {
	catalog := CatalogCopy(StarterCatalog)
	shifted := catalog["brick-2x2"]
	shifted.ID = "shifted-receptors"
	shifted.Connectors = append([]PartConnector(nil), shifted.Connectors...)
	for index := range shifted.Connectors {
		if shifted.Connectors[index].Kind == "receptor" {
			shifted.Connectors[index].X++
		}
	}
	catalog[shifted.ID] = shifted
	report := ValidateWithCatalog([]Placement{
		{ID: "base", PartID: "brick-2x2", Color: 4, Y: 0, Step: 1},
		{ID: "upper", PartID: shifted.ID, Color: 4, Y: 3, Step: 2},
	}, UnlimitedInventory(), catalog)
	if report.Buildable || !hasValidationIssue(report, "unsupported") || !hasValidationIssue(report, "disconnected") {
		t.Fatalf("bbox overlap masqueraded as a connection: %#v", report)
	}
}

func TestRacerWheelPinsUseExactConnectorPairs(t *testing.T) {
	placements := placementsFor(AssemblyRecipe{Archetype: "racer"})
	connections := deriveExactConnections(placements, StarterCatalog)
	wheelConnections := 0
	for _, connection := range connections {
		if connection.Kind == "wheel_pin" {
			wheelConnections++
			if connection.EngagedCount != 1 || connection.CapacityUnits != 2 {
				t.Fatalf("unexpected wheel connection: %#v", connection)
			}
		}
	}
	if wheelConnections != 4 {
		t.Fatalf("wheel connections = %d, want 4: %#v", wheelConnections, connections)
	}
}

func TestStabilityUsesConvexSupportHullInsteadOfBoundingBox(t *testing.T) {
	catalog := CatalogCopy(StarterCatalog)
	heavy := certifiedRectPart(PartSpec{ID: "heavy", StudsX: 2, StudsZ: 2, PlatesY: 30})
	catalog[heavy.ID] = heavy
	stable, margin := componentStabilityMargin([]Placement{
		{ID: "a", PartID: "brick-1x1", X: 0, Y: 0, Z: 0},
		{ID: "b", PartID: "brick-1x1", X: 3, Y: 0, Z: 0},
		{ID: "c", PartID: "brick-1x1", X: 0, Y: 0, Z: 3},
		{ID: "load", PartID: heavy.ID, X: 2, Y: 3, Z: 2},
	}, catalog)
	if stable || margin >= 0 {
		t.Fatalf("COM inside support AABB but outside convex hull was accepted: stable=%v margin=%f", stable, margin)
	}
}

func TestValidateRejectsPartWithoutCertifiedMechanicalSemantics(t *testing.T) {
	catalog := CatalogCopy(StarterCatalog)
	part := catalog["brick-1x1"]
	part.ID = "asset-only"
	part.CertificationLevel = "asset_only"
	part.Connectors = nil
	catalog[part.ID] = part
	report := ValidateWithCatalog([]Placement{{ID: "part", PartID: part.ID, Color: 4, Step: 1}}, UnlimitedInventory(), catalog)
	if report.Buildable || !hasValidationIssue(report, "uncertified_part") {
		t.Fatalf("uncertified part accepted: %#v", report)
	}
}

func hasValidationIssue(report ValidationReport, code string) bool {
	for _, issue := range report.Issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func BenchmarkValidateWithCatalog200PartTower(b *testing.B) {
	placements := make([]Placement, 0, 200)
	for index := 0; index < 200; index++ {
		placements = append(placements, Placement{
			ID: "tower-" + strconv.Itoa(index), PartID: "brick-1x1", Color: 4,
			Y: index * 3, Step: index + 1, Module: "certified-tower",
		})
	}
	if report := ValidateWithCatalog(placements, UnlimitedInventory(), StarterCatalog); !report.Buildable {
		b.Fatalf("benchmark fixture invalid: %#v", report.Issues)
	}
	b.ResetTimer()
	for index := 0; index < b.N; index++ {
		ValidateWithCatalog(placements, UnlimitedInventory(), StarterCatalog)
	}
}
