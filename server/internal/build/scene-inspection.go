package build

import "sort"

type SceneModuleInspection struct {
	Module                  string         `json:"module"`
	Parts                   int            `json:"parts"`
	ConnectionsTo           map[string]int `json:"connections_to"`
	ComponentsAfterRemoving int            `json:"components_after_removing_assigned_parts"`
}

type SceneObjectInspection struct {
	Object        string                  `json:"object"`
	OccupancySize DesignVector            `json:"occupancy_size_studs_plates"`
	Modules       []SceneModuleInspection `json:"assigned_part_groups"`
}

// SceneInspection exposes the compiled part graph rather than interpreting
// construction labels. Group ownership comes from the tiler's attribution;
// a part may span cells of more than one construction. Removing a group means
// excluding its whole assigned parts, not claiming it is a detachable assembly.
func SceneInspection(plan BuildPlan) []SceneObjectInspection {
	byID := map[string]Placement{}
	objects := map[string][]string{}
	edges := map[string][]string{}
	for _, p := range plan.Placements {
		byID[p.ID] = p
		objects[p.Object] = append(objects[p.Object], p.ID)
	}
	for _, c := range plan.Connections {
		a, aOK := byID[c.APlacementID]
		b, bOK := byID[c.BPlacementID]
		if aOK && bOK && a.Object == b.Object {
			edges[a.ID] = append(edges[a.ID], b.ID)
			edges[b.ID] = append(edges[b.ID], a.ID)
		}
	}
	ids := []string{}
	for id := range objects {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := []SceneObjectInspection{}
	for _, id := range ids {
		lo, hi := DesignVector{1000, 1000, 1000}, DesignVector{-1000, -1000, -1000}
		modules := map[string]*SceneModuleInspection{}
		for _, pid := range objects[id] {
			p := byID[pid]
			if modules[p.Module] == nil {
				modules[p.Module] = &SceneModuleInspection{Module: p.Module, ConnectionsTo: map[string]int{}}
			}
			modules[p.Module].Parts++
			for _, other := range edges[pid] {
				if m := byID[other].Module; m != p.Module {
					modules[p.Module].ConnectionsTo[m]++
				}
			}
			size := orientedSizeWith(p, plan.Parts)
			lo = DesignVector{min(lo.X, p.X), min(lo.Y, p.Y), min(lo.Z, p.Z)}
			hi = DesignVector{max(hi.X, p.X+size.x), max(hi.Y, p.Y+plan.Parts[p.PartID].PlatesY), max(hi.Z, p.Z+size.z)}
		}
		groupIDs := []string{}
		for module := range modules {
			groupIDs = append(groupIDs, module)
		}
		sort.Strings(groupIDs)
		inspection := SceneObjectInspection{Object: id, OccupancySize: DesignVector{hi.X - lo.X, hi.Y - lo.Y, hi.Z - lo.Z}, Modules: []SceneModuleInspection{}}
		for _, module := range groupIDs {
			group := modules[module]
			seen := map[string]bool{}
			for _, pid := range objects[id] {
				if byID[pid].Module == module || seen[pid] {
					continue
				}
				group.ComponentsAfterRemoving++
				queue := []string{pid}
				seen[pid] = true
				for n := 0; n < len(queue); n++ {
					for _, other := range edges[queue[n]] {
						if byID[other].Module != module && !seen[other] {
							seen[other] = true
							queue = append(queue, other)
						}
					}
				}
			}
			inspection.Modules = append(inspection.Modules, *group)
		}
		result = append(result, inspection)
	}
	return result
}
