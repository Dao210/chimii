package build

import (
	"fmt"
	"sort"
	"strings"
)

// Face-connected target volume is only a necessary condition. Catalog
// connector validation still decides whether the resulting parts connect.
func checkSceneTargetConnected(target designTarget, fixed []Placement, catalog PartCatalog) error {
	cells := map[DesignVector]string{}
	for p, cell := range target {
		cells[p] = cell.Node
	}
	for _, p := range fixed {
		s := orientedSizeWith(p, catalog)
		for x := 0; x < s.x; x++ {
			for z := 0; z < s.z; z++ {
				for y := 0; y < catalog[p.PartID].PlatesY; y++ {
					q := DesignVector{p.X + x, p.Y + y, p.Z + z}
					if _, exists := cells[q]; exists {
						return designError("target overlaps special construction " + p.Module)
					}
					cells[q] = p.Module
				}
			}
		}
	}
	ordered := make([]DesignVector, 0, len(cells))
	ground := false
	for p := range cells {
		ordered = append(ordered, p)
		ground = ground || p.Y == 0
	}
	if !ground {
		return designError("object has no ground support")
	}
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		if a.Z != b.Z {
			return a.Z < b.Z
		}
		return a.X < b.X
	})
	visited := map[DesignVector]bool{}
	groups := []string{}
	for _, p := range ordered {
		if visited[p] {
			continue
		}
		queue := []DesignVector{p}
		visited[p] = true
		names := map[string]bool{}
		for n := 0; n < len(queue); n++ {
			q := queue[n]
			names[cells[q]] = true
			for _, d := range []DesignVector{{1, 0, 0}, {-1, 0, 0}, {0, 1, 0}, {0, -1, 0}, {0, 0, 1}, {0, 0, -1}} {
				next := DesignVector{q.X + d.X, q.Y + d.Y, q.Z + d.Z}
				if _, exists := cells[next]; exists && !visited[next] {
					visited[next] = true
					queue = append(queue, next)
				}
			}
		}
		labels := []string{}
		for name := range names {
			labels = append(labels, name)
		}
		sort.Strings(labels)
		groups = append(groups, strings.Join(labels, ","))
	}
	if len(groups) > 1 {
		return &BuildError{Code: BuildErrorStructureInvalid, Cause: fmt.Errorf("detached construction volumes: %s; connect them with real supported volume", strings.Join(groups, " | "))}
	}
	return nil
}
