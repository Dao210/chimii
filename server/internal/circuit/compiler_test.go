package circuit

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestReferenceProjects(t *testing.T) {
	c := StarterCatalog()
	for _, p := range c.Projects {
		t.Run(p.ID, func(t *testing.T) {
			d, err := Compile(c, p.ID, "", p.Title.ZH, "project", c.Inventory())
			if err != nil {
				b, _ := json.MarshalIndent(d.Validation, "", "  ")
				t.Fatalf("%v: %s", err, b)
			}
			if d.Validation.PhysicalVerification != "not_tested" {
				t.Fatal("invented physical verification")
			}
			d2, _ := Compile(c, p.ID, "", p.Title.ZH, "project", c.Inventory())
			if d.ContentHash != d2.ContentHash {
				t.Fatal("non-deterministic output")
			}
		})
	}
}

func TestDocumentSnapshotDoesNotFollowLaterEdits(t *testing.T) {
	c := StarterCatalog()
	inv := c.Inventory()
	d, err := Compile(c, "switch-light", "", "Lamp", "project", inv)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(d)
	inv["B1"] = 0
	c.Projects[0].Placements[0].X = 99
	c.Projects[0].Steps[0].Title.EN = "Changed"
	after, _ := json.Marshal(d)
	if string(before) != string(after) {
		t.Fatal("document changed after compile")
	}
}

type circuitTestGenerator struct {
	output string
	err    error
	called bool
}

func (g *circuitTestGenerator) GenerateText(_ context.Context, _ string, system, prompt string) (string, error) {
	g.called = system == plannerPrompt && prompt == "我想听广播"
	return g.output, g.err
}

func TestPlanUsesOnlyConstrainedIntent(t *testing.T) {
	for _, id := range []string{"fm-radio", "unsupported"} {
		g := &circuitTestGenerator{output: `{"project_id":"` + id + `","title":"我的收音机"}`}
		intent, err := Plan(context.Background(), g, "我想听广播")
		if err != nil || intent.ProjectID != id || !g.called {
			t.Fatalf("intent=%+v err=%v", intent, err)
		}
	}
	failure := errors.New("provider failed")
	g := &circuitTestGenerator{err: failure}
	if _, err := Plan(context.Background(), g, "我想听广播"); !errors.Is(err, failure) {
		t.Fatal("provider error was hidden")
	}
	if _, err := Plan(context.Background(), nil, "我想听广播"); err == nil {
		t.Fatal("nil provider was accepted")
	}
}

func TestRejectDamagedCircuits(t *testing.T) {
	for _, name := range []string{"missing speaker", "reverse battery", "wrong layer", "short", "outside", "unknown part", "missing inventory"} {
		t.Run(name, func(t *testing.T) {
			c := StarterCatalog()
			p, _ := c.Project("alarm-sound")
			inv := c.Inventory()
			switch name {
			case "missing speaker":
				p.Placements = append(p.Placements[:3], p.Placements[4:]...)
			case "reverse battery":
				p.Placements[4].Y += 2
				p.Placements[4].Rotation = 180
			case "wrong layer":
				p.Placements[6].Layer = 3
			case "short":
				p.Placements = append(p.Placements, Placement{ID: "short", PartID: "3", X: 4, Y: 2, Rotation: 90, Layer: 3})
			case "outside":
				p.Placements[0].X = 10
			case "unknown part":
				p.Placements[0].PartID = "generic"
			case "missing inventory":
				inv["SP"] = 0
			}
			r := Validate(c, p.ID, p.Placements, inv)
			if r.Passed {
				t.Fatal("accepted invalid circuit")
			}
		})
	}
}

func TestCatalogSnapshotsAreIndependent(t *testing.T) {
	c := StarterCatalog()
	c.Parts[0].Quantity = 0
	if StarterCatalog().Parts[0].Quantity == 0 {
		t.Fatal("mutable shared catalog")
	}
}

func TestIntentRejectsInventedOutputs(t *testing.T) {
	for _, raw := range []string{`{"project_id":"fm-radio","title":"Radio","connections":[]}`, `{"project_id":"bluetooth","title":"Radio"}`, `{"project_id":"fm-radio","title":"Radio"} more`, `{"project_id":"fm-radio","title":""}`} {
		if _, err := ParseIntent(raw); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
}

func TestMissingInventoryIsNotUnlimited(t *testing.T) {
	c := StarterCatalog()
	if _, err := Compile(c, "switch-light", "", "", "project", nil); err == nil {
		t.Fatal("nil inventory accepted")
	}
	r := Validate(c, "switch-light", c.Projects[0].Placements, map[string]int{})
	if r.Passed {
		t.Fatal("empty inventory accepted")
	}
}
