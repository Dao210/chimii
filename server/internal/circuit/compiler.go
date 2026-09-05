package circuit

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

type union map[string]string

func (u union) root(s string) string {
	if u[s] == "" {
		u[s] = s
	}
	if u[s] != s {
		u[s] = u.root(u[s])
	}
	return u[s]
}
func (u union) join(a, b string) { u[u.root(a)] = u.root(b) }

type terminal struct {
	point Point
	layer int
	key   string
}

// Validate checks the reference topology independently of the grid layout.
// Passing means consistency with a documented circuit, not simulated behavior
// or physical certification. General-purpose circuit synthesis is not enabled.
func Validate(c Catalog, projectID string, placements []Placement, inventory map[string]int) Report {
	r := Report{Issues: []Issue{}, Nets: [][]string{}, UsedParts: map[string]int{}, PhysicalVerification: "not_tested"}
	add := func(code, id string) { r.Issues = append(r.Issues, Issue{Code: code, PlacementID: id}) }
	ref, ok := c.Project(projectID)
	if !ok {
		add("unsupported_project", "")
		return r
	}
	if len(placements) > 128 {
		add("too_many_parts", "")
		return r
	}
	for id, n := range inventory {
		p, ok := c.Part(id)
		if !ok || n < 0 || n > p.Quantity {
			r.Issues = append(r.Issues, Issue{Code: "invalid_inventory", PartID: id})
		}
	}
	u := union{}
	points := map[Point][]terminal{}
	byID := map[string]Placement{}
	functional := []string{}
	for _, p := range placements {
		if p.ID == "" {
			add("invalid_id", p.ID)
			continue
		}
		if _, exists := byID[p.ID]; exists {
			add("duplicate_placement", p.ID)
			continue
		}
		byID[p.ID] = p
		part, exists := c.Part(p.PartID)
		if !exists {
			add("unknown_part", p.ID)
			continue
		}
		r.UsedParts[p.PartID]++
		if (p.Rotation != 0 && p.Rotation != 90 && p.Rotation != 180 && p.Rotation != 270) || p.Layer < 1 || p.Layer > 3 {
			add("invalid_placement", p.ID)
			continue
		}
		for _, port := range part.Ports {
			key := p.ID + ":" + port.ID
			point := Transform(p, port.X, port.Y)
			if point.X < 0 || point.X >= c.Columns || point.Y < 0 || point.Y >= c.Rows {
				add("outside_board", p.ID)
			}
			u.root(key)
			points[point] = append(points[point], terminal{point, p.Layer, key})
			if part.Kind != "wire" {
				functional = append(functional, key)
			}
		}
		for _, group := range part.Conductive {
			if len(group) == 0 {
				add("invalid_part_definition", p.ID)
				continue
			}
			for _, port := range group[1:] {
				u.join(p.ID+":"+group[0], p.ID+":"+port)
			}
		}
		for _, v := range part.Body {
			q := Transform(p, v.X, v.Y)
			if q.X < 0 || q.X >= c.Columns || q.Y < 0 || q.Y >= c.Rows {
				add("outside_board", p.ID)
				break
			}
		}
	}
	for _, p := range ref.Placements {
		v, exists := byID[p.ID]
		if !exists || v.PartID != p.PartID {
			add("reference_part_mismatch", p.ID)
		}
	}
	if len(byID) != len(ref.Placements) {
		add("reference_part_mismatch", "")
	}
	for _, terminals := range points {
		for i, a := range terminals {
			for _, b := range terminals[i+1:] {
				if a.layer == b.layer {
					add("snap_collision", strings.Split(b.key, ":")[0])
				} else if abs(a.layer-b.layer) == 1 {
					u.join(a.key, b.key)
				}
			}
		}
	}
	for _, p := range placements {
		part, exists := c.Part(p.PartID)
		if !exists {
			continue
		}
		if p.Layer > 1 {
			supports := 0
			for _, port := range part.Ports {
				q := Transform(p, port.X, port.Y)
				for _, t := range points[q] {
					if t.layer == p.Layer-1 {
						supports++
						break
					}
				}
			}
			if supports < min(2, len(part.Ports)) {
				add("unsupported_layer", p.ID)
			}
		}
	}
	for i, a := range placements {
		pa, ok := c.Part(a.PartID)
		if !ok {
			continue
		}
		for _, b := range placements[i+1:] {
			pb, ok := c.Part(b.PartID)
			if !ok || a.Layer != b.Layer {
				continue
			}
			if bodiesOverlap(a, pa, b, pb, c) {
				add("body_collision", b.ID)
			}
		}
	}
	for id, n := range r.UsedParts {
		if n > inventory[id] {
			r.Issues = append(r.Issues, Issue{Code: "missing_parts", PartID: id, Required: n, Available: inventory[id]})
		}
	}
	groups := map[string][]string{}
	for _, key := range functional {
		root := u.root(key)
		groups[root] = append(groups[root], key)
	}
	for _, g := range groups {
		sort.Strings(g)
		r.Nets = append(r.Nets, g)
	}
	sort.Slice(r.Nets, func(i, j int) bool { return strings.Join(r.Nets[i], "|") < strings.Join(r.Nets[j], "|") })
	expected := map[string]int{}
	for i, g := range ref.ExpectedNets {
		for _, key := range g {
			expected[key] = i
		}
	}
	for i, a := range functional {
		ai, found := expected[a]
		if !found {
			add("unexpected_terminal", strings.Split(a, ":")[0])
			continue
		}
		for _, b := range functional[i+1:] {
			bi, found := expected[b]
			if found && (ai == bi) != (u.root(a) == u.root(b)) {
				add("topology_mismatch", strings.Split(a, ":")[0])
				break
			}
		}
	}
	// Check direct shorts with all switches closed, without treating loads or
	// modules as conductors. This complements the exact reference net comparison.
	shorts := union{}
	for k := range u {
		shorts[k] = u.root(k)
	}
	for _, p := range placements {
		if p.PartID == "S1" {
			shorts.join(p.ID+":a", p.ID+":b")
		}
	}
	for _, p := range placements {
		if p.PartID == "B1" && shorts.root(p.ID+":+") == shorts.root(p.ID+":-") {
			add("power_short", p.ID)
		}
	}
	sort.SliceStable(r.Issues, func(i, j int) bool {
		return r.Issues[i].Code+r.Issues[i].PlacementID+r.Issues[i].PartID < r.Issues[j].Code+r.Issues[j].PlacementID+r.Issues[j].PartID
	})
	r.Passed = len(r.Issues) == 0
	return r
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func bodiesOverlap(a Placement, pa Part, b Placement, pb Part, c Catalog) bool {
	// Quarter-grid interior samples cover the axis-aligned and triangular
	// catalogue shapes. Contact at polygon boundaries is handled by snap checks.
	for y := 0; y < (c.Rows-1)*4; y++ {
		for x := 0; x < (c.Columns-1)*4; x++ {
			fx, fy := float64(x)/4+0.125, float64(y)/4+0.125
			if insideBody(a, pa, fx, fy) && insideBody(b, pb, fx, fy) {
				return true
			}
		}
	}
	return false
}

func insideBody(p Placement, part Part, x, y float64) bool {
	vs := make([]Point, len(part.Body))
	for i, v := range part.Body {
		vs[i] = Transform(p, v.X, v.Y)
	}
	if len(vs) < 3 {
		if len(vs) == 0 {
			return false
		}
		a, b := vs[0], vs[len(vs)-1]
		dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
		den := dx*dx + dy*dy
		t := 0.0
		if den > 0 {
			t = math.Max(0, math.Min(1, ((x-float64(a.X))*dx+(y-float64(a.Y))*dy)/den))
		}
		return math.Hypot(x-float64(a.X)-t*dx, y-float64(a.Y)-t*dy) < 0.18
	}
	inside := false
	j := len(vs) - 1
	for i, a := range vs {
		b := vs[j]
		if (float64(a.Y) > y) != (float64(b.Y) > y) && x < (float64(b.X-a.X)*(y-float64(a.Y))/float64(b.Y-a.Y)+float64(a.X)) {
			inside = !inside
		}
		j = i
	}
	return inside
}

func Compile(c Catalog, projectID, prompt, title, planner string, inventory map[string]int) (Document, error) {
	p, ok := c.Project(projectID)
	if !ok {
		return Document{}, errors.New("unsupported_project")
	}
	if inventory == nil {
		return Document{}, errors.New("inventory_required")
	}
	r := Validate(c, projectID, p.Placements, inventory)
	if !r.Passed {
		return Document{Validation: r}, errors.New("validation_failed")
	}
	if err := validateSteps(p); err != nil {
		return Document{}, err
	}
	parts := []Part{}
	for _, part := range c.Parts {
		if r.UsedParts[part.ID] > 0 {
			parts = append(parts, part)
		}
	}
	d := Document{Version: 1, CatalogVersion: c.Version, KitID: c.KitID, Prompt: prompt, Title: title, Planner: planner, Project: p, Parts: parts, Columns: c.Columns, Rows: c.Rows, Preparation: c.Preparation, Inventory: inventory, Validation: r}
	data, err := json.Marshal(d)
	if err != nil {
		return Document{}, err
	}
	hash := sha256.Sum256(data)
	// Detach nested slices and maps from the caller's catalogue/inventory so
	// subsequent edits cannot silently invalidate this snapshot's content hash.
	var snapshot Document
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Document{}, err
	}
	snapshot.ContentHash = hex.EncodeToString(hash[:])
	return snapshot, nil
}

func validateSteps(p Project) error {
	placements := map[string]Placement{}
	for _, v := range p.Placements {
		placements[v.ID] = v
	}
	seen := map[string]bool{}
	layer := 1
	for _, s := range p.Steps {
		for _, id := range s.PlacementIDs {
			v, ok := placements[id]
			if !ok || seen[id] || v.Layer < layer {
				return fmt.Errorf("invalid step %s", s.ID)
			}
			seen[id] = true
			layer = v.Layer
		}
	}
	if len(seen) != len(placements) {
		return errors.New("incomplete_steps")
	}
	return nil
}
