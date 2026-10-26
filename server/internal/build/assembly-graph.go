package build

import (
	"fmt"
	"sort"
)

// AssemblyInterface is a design contract, not a load-bearing certification.
type AssemblyInterface struct {
	ID                 string `json:"id"`
	A                  string `json:"a"`
	B                  string `json:"b"`
	MinimumConnections int    `json:"minimum_connections"`
}

type InterfaceEvidence struct {
	Object      string   `json:"object"`
	Interface   string   `json:"interface"`
	A           string   `json:"a"`
	B           string   `json:"b"`
	Connections []string `json:"connections"`
	SharedParts []string `json:"shared_parts"`
}

func validateAssemblyGraph(scene SceneSpec) error {
	for _, object := range scene.Objects {
		if len(object.Interfaces) > 128 {
			return designError("too many assembly interfaces: " + object.ID)
		}
		modules := map[string]Construction{}
		for _, m := range object.Modules {
			modules[m.ID] = m
			if m.Role != "" || scene.Version == 2 {
				switch m.Role {
				case "subject", "structure", "detail", "display_support":
				default:
					return designError("invalid construction role: " + object.ID + "/" + m.ID)
				}
			}
		}
		seen, pairs := map[string]bool{}, map[string]bool{}
		adjacency := map[string][]string{}
		for _, contract := range object.Interfaces {
			a, aOK := modules[contract.A]
			b, bOK := modules[contract.B]
			pair := contract.A + ":" + contract.B
			if contract.A > contract.B {
				pair = contract.B + ":" + contract.A
			}
			if !sceneID(contract.ID) || seen[contract.ID] || pairs[pair] || !aOK || !bOK || contract.A == contract.B || a.Kind == "paint" || b.Kind == "paint" || contract.MinimumConnections < 1 || contract.MinimumConnections > 16 {
				return designError("invalid or repeated assembly interface: " + object.ID + "/" + contract.ID)
			}
			seen[contract.ID], pairs[pair] = true, true
			if a.Role != "display_support" && b.Role != "display_support" {
				adjacency[a.ID] = append(adjacency[a.ID], b.ID)
				adjacency[b.ID] = append(adjacency[b.ID], a.ID)
			}
		}
		if scene.Version != 2 {
			continue
		}
		root := ""
		for _, m := range object.Modules {
			if m.Role == "subject" && m.Kind != "paint" {
				root = m.ID
				break
			}
		}
		if root == "" {
			return designError("scene object needs a subject construction: " + object.ID)
		}
		reachable := map[string]bool{root: true}
		queue := []string{root}
		for n := 0; n < len(queue); n++ {
			for _, next := range adjacency[queue[n]] {
				if !reachable[next] {
					reachable[next] = true
					queue = append(queue, next)
				}
			}
		}
		for _, m := range object.Modules {
			if m.Role == "subject" && m.Kind != "paint" && !reachable[m.ID] {
				return designError("subject interface graph relies only on display support: " + object.ID + "/" + m.ID)
			}
		}
	}
	return nil
}

func inspectAssemblyInterfaces(plan BuildPlan, scene SceneSpec, catalog PartCatalog) ([]InterfaceEvidence, error) {
	result := []InterfaceEvidence{}
	for _, object := range scene.Objects {
		if len(object.Interfaces) == 0 {
			continue
		}
		target, _, err := sceneObjectTarget(object, catalog)
		if err != nil {
			return nil, err
		}
		membership := map[string]map[string]bool{}
		ids := []string{}
		for _, p := range plan.Placements {
			if p.Object != object.ID {
				continue
			}
			ids = append(ids, p.ID)
			members := map[string]bool{p.Module: true}
			size := orientedSizeWith(p, catalog)
			for x := 0; x < size.x; x++ {
				for y := 0; y < catalog[p.PartID].PlatesY; y++ {
					for z := 0; z < size.z; z++ {
						if cell, ok := target[DesignVector{p.X + x, p.Y + y, p.Z + z}]; ok {
							members[cell.Node] = true
						}
					}
				}
			}
			membership[p.ID] = members
		}
		sort.Strings(ids)
		for _, contract := range object.Interfaces {
			a, b := object.ID+"/"+contract.A, object.ID+"/"+contract.B
			e := InterfaceEvidence{Object: object.ID, Interface: contract.ID, A: contract.A, B: contract.B, Connections: []string{}, SharedParts: []string{}}
			for _, id := range ids {
				if membership[id][a] && membership[id][b] {
					e.SharedParts = append(e.SharedParts, id)
				}
			}
			for _, connection := range plan.Connections {
				left, right := membership[connection.APlacementID], membership[connection.BPlacementID]
				// A shared brick already supplies one bridge. Its contacts to
				// either side must not count that same bridge a second time.
				if left[a] && !left[b] && right[b] && !right[a] || left[b] && !left[a] && right[a] && !right[b] {
					e.Connections = append(e.Connections, connection.ID)
				}
			}
			sort.Strings(e.Connections)
			if len(e.Connections)+len(e.SharedParts) < contract.MinimumConnections {
				return result, &BuildError{Code: BuildErrorRequirements, Cause: fmt.Errorf("assembly interface %s/%s requires %d actual connections or bridging parts between %s and %s; found %d connections and %d bridging parts", object.ID, contract.ID, contract.MinimumConnections, contract.A, contract.B, len(e.Connections), len(e.SharedParts))}
			}
			result = append(result, e)
		}
	}
	return result, nil
}
