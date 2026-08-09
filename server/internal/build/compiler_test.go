package build

import (
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

func TestQuestionForSkipsAlreadySpecificPrompt(t *testing.T) {
	if QuestionFor("一辆会跑的红色小车") != nil {
		t.Fatal("specific prompt should proceed")
	}
	if QuestionFor("做一只小狗") == nil {
		t.Fatal("ambiguous prompt should ask one question")
	}
}

func TestRacerUsesOfficialCompatibleWheelHolderAndCenteredLDrawCoordinates(t *testing.T) {
	result, err := Compile(PlanRecipe("一辆会跑的红色小车", nil), UnlimitedInventory(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"4600.dat", "4624c04.dat", "1 4 20 -8 40"} {
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
