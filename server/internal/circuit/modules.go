package circuit

import (
	"sort"
	"strings"
)

// ValidateModuleConnections checks a reviewed keyed-module assembly. It does
// not simulate analogue behaviour or infer an undocumented module's internals.
func ValidateModuleConnections(c Catalog, projectID string, placements []Placement, connections []Connection, inventory map[string]int) Report {
	r := Report{Issues: []Issue{}, Nets: [][]string{}, UsedParts: map[string]int{}, PhysicalVerification: "not_tested"}
	add := func(code, id string) { r.Issues = append(r.Issues, Issue{Code: code, PlacementID: id}) }
	ref, ok := c.Project(projectID)
	if !ok || c.ConnectionSystem != "boson" {
		add("unsupported_project", "")
		return r
	}
	if len(placements) > 128 || len(connections) > 128 {
		add("too_many_parts", "")
		return r
	}
	for id, n := range inventory {
		p, exists := c.Part(id)
		if !exists || n < 0 || n > p.Quantity {
			r.Issues = append(r.Issues, Issue{Code: "invalid_inventory", PartID: id})
		}
	}
	byID := map[string]Placement{}
	ports := map[string]Port{}
	for _, p := range placements {
		if _, exists := byID[p.ID]; exists || p.ID == "" {
			add("duplicate_placement", p.ID)
			continue
		}
		byID[p.ID] = p
		part, exists := c.Part(p.PartID)
		if !exists {
			add("unknown_part", p.ID)
			continue
		}
		r.UsedParts[part.ID]++
		if p.Rotation != 0 || p.Layer != 1 || p.X < 0 || p.Y < 0 || p.X >= c.Columns || p.Y >= c.Rows {
			add("invalid_placement", p.ID)
		}
		if part.Kind == "cable" {
			continue
		}
		for _, port := range part.Ports {
			key := p.ID + ":" + port.ID
			if _, exists := ports[key]; exists || port.Connector == "" {
				add("invalid_part_definition", p.ID)
			}
			ports[key] = port
		}
	}
	if len(byID) != len(ref.Placements) {
		add("reference_part_mismatch", "")
	}
	for _, p := range ref.Placements {
		if actual, exists := byID[p.ID]; !exists || actual.PartID != p.PartID {
			add("reference_part_mismatch", p.ID)
		}
	}
	usedPorts := map[string]bool{}
	usedCables := map[string]bool{}
	usedIDs := map[string]bool{}
	u := union{}
	for id, p := range byID {
		if part, exists := c.Part(p.PartID); exists && part.Kind != "cable" {
			u.root(id)
		}
	}
	for _, wire := range connections {
		if wire.ID == "" || usedIDs[wire.ID] {
			add("duplicate_connection", wire.ID)
		}
		usedIDs[wire.ID] = true
		from, fromOK := ports[wire.From]
		to, toOK := ports[wire.To]
		if !fromOK || !toOK {
			add("unknown_port", wire.ID)
			continue
		}
		if wire.From == wire.To || usedPorts[wire.From] || usedPorts[wire.To] {
			add("port_reused", wire.ID)
		}
		usedPorts[wire.From] = true
		usedPorts[wire.To] = true
		if from.Connector != to.Connector {
			add("connector_mismatch", wire.ID)
		}
		if !((from.Direction == "output" && to.Direction == "input") || (from.Direction == "power_source" && to.Direction == "power_sink")) {
			add("signal_direction", wire.ID)
		}
		if from.Connector == "boson-ph2-3" {
			cable, exists := byID[wire.CableID]
			part, known := c.Part(cable.PartID)
			if !exists || !known || part.Kind != "cable" || usedCables[wire.CableID] {
				add("invalid_cable", wire.ID)
			}
			usedCables[wire.CableID] = true
		} else if from.Connector != "fit0529-usb" || wire.CableID != "" {
			add("invalid_power_connection", wire.ID)
		}
		u.join(strings.Split(wire.From, ":")[0], strings.Split(wire.To, ":")[0])
		net := []string{wire.From, wire.To}
		sort.Strings(net)
		r.Nets = append(r.Nets, net)
	}
	for key := range ports {
		if !usedPorts[key] {
			add("unconnected_port", key)
		}
	}
	for id, p := range byID {
		part, exists := c.Part(p.PartID)
		if exists && part.Kind == "cable" && !usedCables[id] {
			add("unused_cable", id)
		}
	}
	roots := map[string]bool{}
	for id := range u {
		roots[u.root(id)] = true
	}
	if len(roots) != 1 {
		add("disconnected_assembly", "")
	}
	if moduleNetKey(ref.ExpectedNets) != moduleNetKey(r.Nets) {
		add("reference_net_mismatch", "")
	}
	for id, count := range r.UsedParts {
		if inventory[id] < count {
			r.Issues = append(r.Issues, Issue{Code: "missing_part", PartID: id, Required: count, Available: inventory[id]})
		}
	}
	sort.Slice(r.Nets, func(i, j int) bool { return strings.Join(r.Nets[i], "|") < strings.Join(r.Nets[j], "|") })
	sort.Slice(r.Issues, func(i, j int) bool {
		a, b := r.Issues[i], r.Issues[j]
		return a.Code+a.PlacementID+a.PartID < b.Code+b.PlacementID+b.PartID
	})
	r.Passed = len(r.Issues) == 0
	return r
}

func moduleNetKey(nets [][]string) string {
	keys := make([]string, 0, len(nets))
	for _, net := range nets {
		copy := append([]string{}, net...)
		sort.Strings(copy)
		keys = append(keys, strings.Join(copy, "|"))
	}
	sort.Strings(keys)
	return strings.Join(keys, "\n")
}
