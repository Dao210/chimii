package build

import (
	"fmt"
	"sort"
)

const MaxRecipeModules = 24
const BuildErrorUnsupported = "BUILD_UNSUPPORTED"
const BuildErrorRequirements = "BUILD_REQUIREMENTS_UNMET"

type modulePort struct {
	X, Y, Z int      `json:"-"`
	Accepts []string `json:"accepts"`
}
type moduleDefinition struct {
	Kind        string                `json:"kind"`
	Description string                `json:"description"`
	Root        bool                  `json:"root"`
	Ports       map[string]modulePort `json:"ports"`
	PartCount   int                   `json:"part_count"`
	Parts       []Placement           `json:"-"`
}

// The registry is the single source of planner capabilities and compiler geometry.
// Ports constrain composition; every assembled model still passes full mechanics checks.
var moduleRegistry = makeModuleRegistry()

func makeModuleRegistry() map[string]moduleDefinition {
	part := func(id string, x, y, z, rotation, step int) Placement {
		return Placement{PartID: id, X: x, Y: y, Z: z, Rotation: rotation, Step: step}
	}
	top := func(x, y, z int, kinds ...string) modulePort { return modulePort{x, y, z, kinds} }
	r := map[string]moduleDefinition{}
	add := func(kind, description string, root bool, ports map[string]modulePort, parts ...Placement) {
		r[kind] = moduleDefinition{kind, description, root, ports, len(parts), parts}
	}
	add("body", "Horizontal body, for an animal or simple structure", true, map[string]modulePort{
		"front": top(0, 3, 0, "head"), "center": top(1, 3, 0, "head", "roof"),
		"left": top(-3, 3, 0, "wing"), "right": top(3, 3, 0, "wing", "tail"),
		"stack": top(0, 3, 0, "body"),
	}, part("brick-2x4", 0, 0, 0, 0, 1))
	add("head", "Square head with optional face and ears", false, map[string]modulePort{
		"face": top(0, 3, 0, "eyes"), "ears": top(0, 3, 1, "short-ears", "long-ears"), "roof": top(0, 3, 0, "roof"),
	}, part("brick-2x2", 0, 0, 0, 0, 1))
	add("eyes", "Two raised eyes", false, nil, part("brick-1x1", 0, 0, 0, 0, 1), part("brick-1x1", 1, 0, 0, 0, 1))
	add("short-ears", "Two short static ears", false, nil, part("brick-1x1", 0, 0, 0, 0, 1), part("brick-1x1", 1, 0, 0, 0, 1))
	add("long-ears", "Two tall static ears, not articulated", false, nil, part("brick-1x1", 0, 0, 0, 0, 1), part("brick-1x1", 1, 0, 0, 0, 1), part("brick-1x1", 0, 3, 0, 0, 2), part("brick-1x1", 1, 3, 0, 0, 2))
	add("wing", "Static horizontal wing; no real flight or flapping", false, nil, part("plate-2x4", 0, 0, 0, 0, 1))
	add("tail", "Static tail silhouette, not a moving joint", false, nil, part("plate-1x2", 0, 0, 0, 0, 1), part("slope-2x2", 0, 1, 0, 0, 2))
	add("roof", "Sloped roof or cabin top", false, nil, part("slope-2x2", 0, 0, 0, 0, 1))
	add("robot-base", "Two feet and upright torso; no articulated arms", true, map[string]modulePort{"head": top(1, 9, 0, "head")}, part("brick-2x2", 0, 0, 0, 0, 1), part("brick-2x2", 2, 0, 0, 0, 1), part("brick-2x4", 0, 3, 0, 0, 2), part("brick-2x4", 0, 6, 0, 0, 3))
	add("rolling-base", "Vehicle base with four real rolling wheels", true, map[string]modulePort{"cabin": top(0, 5, 1, "head")},
		part("wheel-holder-2x2", 0, 0, 0, 0, 1), part("wheel-holder-2x2", 0, 0, 2, 0, 1), part("plate-2x4", 0, 1, 0, 90, 2),
		part("wheel", -1, 0, 0, 90, 3), part("wheel", 2, 0, 0, 270, 3), part("wheel", -1, 0, 2, 90, 3), part("wheel", 2, 0, 2, 270, 3), part("brick-2x4", 0, 2, 0, 90, 4))
	return r
}

// Capabilities uses BOM filtering, not a full mechanics solve per candidate.
// Configured quantities are checked globally again after composition.
func Capabilities(inventory InventorySnapshot, catalog PartCatalog) []moduleDefinition {
	counts := map[string]int{}
	for _, i := range inventory.Items {
		counts[i.PartID] += i.Quantity
	}
	result := []moduleDefinition{}
	for _, d := range moduleRegistry {
		needed := map[string]int{}
		ok := true
		for _, p := range d.Parts {
			needed[p.PartID]++
			spec, exists := catalog[p.PartID]
			if !exists || !partIsMechanicallyCertified(spec) {
				ok = false
			}
		}
		if inventory.Configured {
			for id, n := range needed {
				if counts[id] < n {
					ok = false
				}
			}
		}
		if ok {
			result = append(result, d)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Kind < result[j].Kind })
	return result
}

func ExpandRecipe(recipe AssemblyRecipe) ([]Placement, error) {
	fail := func(message string) ([]Placement, error) {
		return nil, &BuildError{Code: BuildErrorRequirements, Cause: fmt.Errorf("%s", message)}
	}
	if recipe.Version != 2 || len(recipe.Modules) == 0 || len(recipe.Modules) > MaxRecipeModules {
		return fail("invalid module recipe")
	}
	type placedModule struct {
		Definition        moduleDefinition
		X, Y, Z, LastStep int
	}
	placed := map[string]placedModule{}
	occupiedPorts := map[string]bool{}
	kinds := map[string]bool{}
	result := []Placement{}
	step := 0
	for index, m := range recipe.Modules {
		d, ok := moduleRegistry[m.Kind]
		if !ok || m.ID == "" || len(m.ID) > 48 || !IsAllowedColor(m.Color) {
			return fail("unknown module, id or color")
		}
		if _, exists := placed[m.ID]; exists {
			return fail("duplicate module id")
		}
		x, y, z := 0, 0, 0
		if index == 0 {
			if !d.Root || m.Parent != "" || m.Port != "" {
				return fail("first module must be a root")
			}
		} else {
			parent, exists := placed[m.Parent]
			if !exists {
				return fail("parent must precede child")
			}
			port, exists := parent.Definition.Ports[m.Port]
			allowed := false
			for _, kind := range port.Accepts {
				if kind == m.Kind {
					allowed = true
				}
			}
			key := m.Parent + "/" + m.Port
			if len(m.AlternativePorts) > 2 {
				return fail("too many alternative ports")
			}
			for _, alt := range m.AlternativePorts {
				p, exists := parent.Definition.Ports[alt]
				ok := false
				for _, kind := range p.Accepts {
					if kind == m.Kind {
						ok = true
					}
				}
				if !exists || !ok {
					return fail("invalid alternative port")
				}
			}
			if !exists || !allowed || occupiedPorts[key] {
				return fail("invalid or occupied module port")
			}
			occupiedPorts[key] = true
			x, y, z = parent.X+port.X, parent.Y+port.Y, parent.Z+port.Z
		}
		if recipe.Constraints.NoWheels && m.Kind == "rolling-base" {
			return fail("wheels violate the request")
		}
		last := step
		for _, local := range d.Parts {
			p := local
			p.ID = fmt.Sprintf("%s-%d", m.ID, len(result)+1)
			p.Module = m.Kind
			p.X += x
			p.Y += y
			p.Z += z
			p.Step += step
			p.Color = m.Color
			// Mechanical wheel assemblies retain their material color only when color is flexible.
			if !recipe.Constraints.ExactColors && (p.PartID == "wheel" || p.PartID == "wheel-holder-2x2") {
				p.Color = 71
			}
			result = append(result, p)
			if p.Step > last {
				last = p.Step
			}
		}
		step = last
		placed[m.ID] = placedModule{d, x, y, z, last}
		kinds[m.Kind] = true
	}
	if len(result) > 200 {
		return fail("part budget exceeded")
	}
	if recipe.Constraints.PartCount > 0 && recipe.Constraints.PartCount != len(result) {
		return nil, &BuildError{Code: BuildErrorCountUnsupported, Cause: fmt.Errorf("requested %d parts, composed %d", recipe.Constraints.PartCount, len(result))}
	}
	for _, kind := range recipe.Constraints.RequiredModules {
		if !kinds[kind] {
			return fail("missing required module: " + kind)
		}
	}
	return result, nil
}

// ExampleRecipe is a compact planner example, compiled by the same registry as new designs.
func ExampleRecipe(subject string) AssemblyRecipe {
	m := func(id, kind, parent, port string, color int) ModuleInstance {
		return ModuleInstance{ID: id, Kind: kind, Parent: parent, Port: port, Color: color}
	}
	r := AssemblyRecipe{Version: 2, Subject: subject, Archetype: subject, Title: subject, Palette: []int{4, 14, 1, 15}, Features: []string{}, Metadata: map[string]string{}}
	switch subject {
	case "racer":
		r.Modules = []ModuleInstance{m("base", "rolling-base", "", "", 4), m("head", "head", "base", "cabin", 1), m("roof", "roof", "head", "roof", 15)}
	case "robot":
		r.Modules = []ModuleInstance{m("base", "robot-base", "", "", 14), m("head", "head", "base", "head", 1), m("eyes", "eyes", "head", "face", 15)}
	case "flyer":
		r.Modules = []ModuleInstance{m("body", "body", "", "", 1), m("head", "head", "body", "center", 14), m("left", "wing", "body", "left", 4), m("right", "wing", "body", "right", 4)}
	default:
		r.Modules = []ModuleInstance{m("body", "body", "", "", 2), m("head", "head", "body", "front", 14), m("eyes", "eyes", "head", "face", 15), m("tail", "tail", "body", "right", 4)}
	}
	return r
}
