package circuit

import (
	"encoding/json"
	"testing"
)

func TestCompositionGeneratesNewGraphsAndIndependentBehavior(t *testing.T) {
	c := bosonCatalog(t)
	for _, output := range []string{"BOS0017-R", "BOS0021"} {
		for _, op := range []string{"direct", "not", "and"} {
			for _, input := range []string{"BOS0002-R", "BOS0013"} {
				spec := CompositionSpec{Inputs: []string{input}, Operation: op, Output: output}
				if op == "and" {
					spec.Inputs = []string{"BOS0002-R", "BOS0013"}
				}
				d, err := CompileComposition(c, spec, "new idea", "New creation", c.Inventory())
				if err != nil {
					t.Fatalf("%+v: %v, %+v", spec, err, d.Validation)
				}
				if _, exists := c.Project(d.Project.ID); exists {
					t.Fatal("selected a fixed project")
				}
				if !d.Behavior.Passed || len(d.Behavior.Cases) != (1<<len(spec.Inputs))+1 || d.Validation.PhysicalVerification != "not_tested" {
					t.Fatal("wrong verification evidence")
				}
				if d.Project.Steps[len(d.Project.Steps)-1].PlacementIDs[0] != "battery" {
					t.Fatal("power not last")
				}
				last := d.Behavior.Cases[len(d.Behavior.Cases)-1]
				if last.Powered || last.Actual || last.Expected {
					t.Fatal("unpowered output active")
				}
				if op == "and" {
					break
				}
			}
		}
	}
}

func TestBehaviorDetectsIncorrectImplementationDespiteMatchingNetlist(t *testing.T) {
	c := bosonCatalog(t)
	wanted := CompositionSpec{Inputs: []string{"BOS0002-R"}, Operation: "not", Output: "BOS0017-R"}
	wrong, _ := ComposeProject(c, CompositionSpec{Inputs: wanted.Inputs, Operation: "direct", Output: wanted.Output})
	if !validateModuleGraph(c, nil, wrong.Placements, wrong.Connections, c.Inventory()).Passed {
		t.Fatal("structurally valid baseline failed")
	}
	if CheckCompositionBehavior(c, wanted, wrong).Passed {
		t.Fatal("missing inversion passed functional check")
	}
	wanted.Operation = "and"
	wanted.Inputs = append(wanted.Inputs, "BOS0013")
	if CheckCompositionBehavior(c, wanted, wrong).Passed {
		t.Fatal("dropped condition passed")
	}
}

func TestCompositionRejectsInventoryAndGraphDamage(t *testing.T) {
	c := bosonCatalog(t)
	spec := CompositionSpec{Inputs: []string{"BOS0002-R", "BOS0013"}, Operation: "and", Output: "BOS0021"}
	for _, defect := range []string{"missing cable", "reversed", "power", "cycle", "unknown part"} {
		t.Run(defect, func(t *testing.T) {
			p, _ := ComposeProject(c, spec)
			inv := c.Inventory()
			switch defect {
			case "missing cable":
				inv["BOSON-CABLE-10CM"] = 3
			case "reversed":
				p.Connections[0].From, p.Connections[0].To = p.Connections[0].To, p.Connections[0].From
			case "power":
				p.Connections = p.Connections[:len(p.Connections)-1]
			case "cycle":
				p.Connections[0].From = "logic:OUT"
			case "unknown part":
				p.Placements[0].PartID = "unreviewed-power"
			}
			if validateModuleGraph(c, nil, p.Placements, p.Connections, inv).Passed {
				t.Fatal("accepted damaged composition")
			}
		})
	}
	inv := c.Inventory()
	inv[spec.Output] = 0
	if d, err := CompileComposition(c, spec, "", "", inv); err == nil || d.Validation.Passed {
		t.Fatal("ignored missing output")
	}
}

func TestCompositionPlannerBoundaryAndSnapshot(t *testing.T) {
	c := bosonCatalog(t)
	valid := `{"outcome":"ready","title":"按钮风扇","composition":{"inputs":["BOS0002-R"],"operation":"direct","output":"BOS0021"}}`
	d, err := ParseConversationDecision(valid, c)
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{
		`{"outcome":"ready","title":"fan","project_id":"boson-button-light","composition":{"inputs":["BOS0002-R"],"operation":"direct","output":"BOS0021"}}`,
		`{"outcome":"ready","title":"fan","composition":{"inputs":["BOS0002-R"],"operation":"toggle","output":"BOS0021"}}`,
		`{"outcome":"ready","title":"fan","composition":{"inputs":["BOS0002-R","BOS0002-R"],"operation":"and","output":"BOS0021"}}`,
		`{"outcome":"ready","title":"fan","composition":{"inputs":["BOS0009"],"operation":"direct","output":"BOS0021"}}`,
		`{"outcome":"clarify","question":"which?","composition":{"inputs":["BOS0002-R"],"operation":"direct","output":"BOS0021"}}`,
		`{"outcome":"ready","title":"fan","composition":{"inputs":["BOS0002-R"],"operation":"direct","output":"BOS0021","connections":[]}}`,
	} {
		if _, err := ParseConversationDecision(raw, c); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := ParseConversationDecision(valid, StarterCatalog()); err == nil {
		t.Fatal("cross-kit composition")
	}
	doc, err := CompileComposition(c, *d.Composition, "", "Fan", c.Inventory())
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(doc)
	d.Composition.Inputs[0] = "modified"
	c.Parts[0].Ports[0].ID = "bad"
	after, _ := json.Marshal(doc)
	if string(before) != string(after) {
		t.Fatal("snapshot was mutated")
	}
}
