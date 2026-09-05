package circuit

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func bosonCatalog(t *testing.T) Catalog {
	t.Helper()
	c, ok := FindCatalog("dfrobot-edu0080-en")
	if !ok {
		t.Fatal("missing BOSON catalogue")
	}
	return c
}

func TestBOSONReferenceProjects(t *testing.T) {
	c := bosonCatalog(t)
	if len(c.Parts) != 11 || len(c.Projects) != 5 {
		t.Fatal("wrong documented subset")
	}
	for _, p := range c.Projects {
		t.Run(p.ID, func(t *testing.T) {
			d, err := Compile(c, p.ID, "", p.Title.EN, "project", c.Inventory())
			if err != nil {
				t.Fatalf("%v: %+v", err, d.Validation)
			}
			if d.Version != 2 || d.ConnectionSystem != "boson" || d.Validation.PhysicalVerification != "not_tested" || !d.Validation.Passed {
				t.Fatalf("bad evidence: %+v", d)
			}
			if len(d.Project.Connections) != len(d.Project.ExpectedNets) {
				t.Fatal("incomplete connections")
			}
			last := d.Project.Steps[len(d.Project.Steps)-1]
			if len(last.PlacementIDs) != 1 || last.PlacementIDs[0] != "battery" {
				t.Fatal("power must be connected last")
			}
		})
	}
}

func TestBOSONRejectsDamagedModuleGraphs(t *testing.T) {
	for _, defect := range []string{"reversed", "same-port", "missing-cable", "reused-cable", "unknown-port", "extra-wire", "missing-part", "cross-brand", "wrong-power-port"} {
		t.Run(defect, func(t *testing.T) {
			c := bosonCatalog(t)
			p := c.Projects[4]
			w := p.Connections
			inv := c.Inventory()
			switch defect {
			case "reversed":
				w[0].From, w[0].To = w[0].To, w[0].From
			case "same-port":
				w[1].To = w[0].To
			case "missing-cable":
				w[0].CableID = ""
			case "reused-cable":
				w[1].CableID = w[0].CableID
			case "unknown-port":
				w[0].From = "motion:MAGIC"
			case "extra-wire":
				w = append(w, w[0])
			case "missing-part":
				inv["BOSON-CABLE-10CM"] = 3
			case "cross-brand":
				p.Placements[0].PartID = "B1"
			case "wrong-power-port":
				w[len(w)-1].To = "power:IN"
			}
			if got := ValidateModuleConnections(c, p.ID, p.Placements, w, inv); got.Passed {
				t.Fatalf("accepted %s", defect)
			}
		})
	}
}

func TestBOSONSnapshotAndCatalogueIsolation(t *testing.T) {
	c := bosonCatalog(t)
	inventory := c.Inventory()
	d, err := Compile(c, c.Projects[0].ID, "", "Lamp", "project", inventory)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(d)
	c.Parts[0].Ports[0].Connector = "bad"
	c.Projects[0].Connections[0].To = "bad"
	inventory["BOS0036"] = 0
	after, _ := json.Marshal(d)
	if string(before) != string(after) {
		t.Fatal("saved document mutated")
	}
	fresh := bosonCatalog(t)
	if fresh.Parts[0].Ports[0].Connector == "bad" {
		t.Fatal("shared catalogue mutated")
	}
}

type bosonGenerator struct {
	prompt string
	output string
}

func (g *bosonGenerator) GenerateText(_ context.Context, _ string, prompt, _ string) (string, error) {
	g.prompt = prompt
	return g.output, nil
}
func TestBOSONPlannerCannotSelectAnotherKitOrInventWiring(t *testing.T) {
	c := bosonCatalog(t)
	for _, raw := range []string{`{"project_id":"fm-radio","title":"Radio"}`, `{"project_id":"boson-sound-fan","title":"Fan","connections":[]}`} {
		g := &bosonGenerator{output: raw}
		if _, err := PlanForCatalog(context.Background(), g, c, "Make a fan"); err == nil {
			t.Fatal("unbounded planner output accepted")
		}
	}
	g := &bosonGenerator{output: `{"project_id":"boson-sound-fan","title":"Fan"}`}
	if _, err := PlanForCatalog(context.Background(), g, c, "Make a fan"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(g.prompt, "not speech recognition") || !strings.Contains(g.prompt, "boson-motion-sound-fan") {
		t.Fatal("missing limits or project authority")
	}
}
