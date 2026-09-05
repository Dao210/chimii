package build

import (
	"testing"
	"time"
)

func TestModuleVariantsChangeGeometryAndPreserveConstraints(t *testing.T) {
	r := ExampleRecipe("creature")
	r.Modules = append(r.Modules, ModuleInstance{ID: "ears", Kind: "short-ears", Parent: "head", Port: "ears", Color: 1})
	first, err := Compile(r, UnlimitedInventory(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	r.Modules[len(r.Modules)-1].Kind = "long-ears"
	r.Constraints = RecipeConstraints{NoWheels: true, ExactColors: true, RequiredModules: []string{"long-ears"}}
	second, err := Compile(r, UnlimitedInventory(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if first.Plan.ContentHash == second.Plan.ContentHash || second.Plan.Validation.PartCount != first.Plan.Validation.PartCount+2 {
		t.Fatal("ear choice did not change geometry")
	}
	r.Modules[len(r.Modules)-1].Kind = "short-ears"
	if _, err = Compile(r, UnlimitedInventory(), time.Unix(0, 0)); err == nil {
		t.Fatal("required long ears were dropped")
	}
}
func TestCompileDoesNotReplaceExplicitColorsOrWheels(t *testing.T) {
	r := ExampleRecipe("creature")
	p, _ := ExpandRecipe(r)
	counts := map[string]int{}
	for _, part := range p {
		counts[part.PartID]++
	}
	items := []InventoryItem{}
	for id, n := range counts {
		items = append(items, InventoryItem{PartID: id, Color: 1, Quantity: n})
	}
	inv := NewInventorySnapshot(true, 1, items)
	if _, err := Compile(r, inv, time.Unix(0, 0)); err != nil {
		t.Fatal(err)
	}
	r.Constraints.ExactColors = true
	if _, err := Compile(r, inv, time.Unix(0, 0)); err == nil {
		t.Fatal("silently replaced explicit colors")
	}
	r = ExampleRecipe("racer")
	r.Constraints.NoWheels = true
	if _, err := Compile(r, UnlimitedInventory(), time.Unix(0, 0)); err == nil {
		t.Fatal("ignored no wheels")
	}
}
func TestCompilerRepairsOnlyPermittedLayoutAlternatives(t *testing.T) {
	r := ExampleRecipe("flyer")
	r.Modules[1].Port = "front"
	if _, err := Compile(r, UnlimitedInventory(), time.Unix(0, 0)); err == nil {
		t.Fatal("unpermitted layout repair")
	}
	r.Modules[1].AlternativePorts = []string{"center"}
	result, err := Compile(r, UnlimitedInventory(), time.Unix(0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if result.Recipe.Modules[1].Port != "center" {
		t.Fatal("repaired recipe not saved")
	}
}
func TestModuleRegistryRejectsCyclesUnknownPortsAndPartBudgets(t *testing.T) {
	for _, mutate := range []func(*AssemblyRecipe){
		func(r *AssemblyRecipe) { r.Modules[1].Parent = r.Modules[1].ID }, func(r *AssemblyRecipe) { r.Modules[1].Port = "imaginary" }, func(r *AssemblyRecipe) { r.Modules[1].AlternativePorts = []string{"imaginary"} }, func(r *AssemblyRecipe) { r.Modules[1].ID = r.Modules[0].ID }, func(r *AssemblyRecipe) { r.Constraints.PartCount = 200 },
	} {
		r := ExampleRecipe("robot")
		mutate(&r)
		if _, err := ExpandRecipe(r); err == nil {
			t.Fatal("invalid graph accepted")
		}
	}
}
func BenchmarkModuleCompile(b *testing.B) {
	r := ExampleRecipe("creature")
	i := UnlimitedInventory()
	now := time.Unix(0, 0)
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		if _, err := Compile(r, i, now); err != nil {
			b.Fatal(err)
		}
	}
}
func BenchmarkModuleCapabilityFilter(b *testing.B) {
	i := UnlimitedInventory()
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		Capabilities(i, StarterCatalog)
	}
}
