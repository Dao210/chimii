package build

import (
	"context"
	"fmt"
	"sort"
)

type constructionBounds struct{ min, max DesignVector }

type constructionVolume struct {
	cells  []DesignVector
	bounds constructionBounds
}

var paintMarkNeighbors = func() []DesignVector {
	offsets := []DesignVector{}
	for x := -1; x <= 1; x++ {
		for y := -1; y <= 1; y++ {
			for z := -1; z <= 1; z++ {
				if x != 0 || y != 0 || z != 0 {
					offsets = append(offsets, DesignVector{x, y, z})
				}
			}
		}
	}
	return offsets
}()

func moduleVolume(module Construction, catalog PartCatalog) (constructionVolume, error) {
	cells, fixed, err := constructionCells(module)
	if err != nil {
		return constructionVolume{}, err
	}
	volume := constructionVolume{bounds: constructionBounds{min: DesignVector{1000, 1000, 1000}, max: DesignVector{-1000, -1000, -1000}}}
	seen := map[DesignVector]bool{}
	add := func(p DesignVector) {
		q := transformCell(p, module.Position, module.Rotation)
		if seen[q] {
			return
		}
		seen[q] = true
		volume.cells = append(volume.cells, q)
		volume.bounds.min = DesignVector{min(volume.bounds.min.X, q.X), min(volume.bounds.min.Y, q.Y), min(volume.bounds.min.Z, q.Z)}
		volume.bounds.max = DesignVector{max(volume.bounds.max.X, q.X), max(volume.bounds.max.Y, q.Y), max(volume.bounds.max.Z, q.Z)}
	}
	for p := range cells {
		add(p)
	}
	for _, p := range fixed {
		spec, exists := catalog[p.PartID]
		if !exists {
			return constructionVolume{}, designError("special construction part absent from catalog: " + p.PartID)
		}
		size := orientedSizeWith(p, catalog)
		for x := 0; x < size.x; x++ {
			for z := 0; z < size.z; z++ {
				for y := 0; y < spec.PlatesY; y++ {
					add(DesignVector{p.X + x, p.Y + y, p.Z + z})
				}
			}
		}
	}
	if len(volume.cells) == 0 {
		return volume, designError("empty construction bounds")
	}
	sort.Slice(volume.cells, func(i, j int) bool {
		a, b := volume.cells[i], volume.cells[j]
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		if a.Z != b.Z {
			return a.Z < b.Z
		}
		return a.X < b.X
	})
	return volume, nil
}

func translatedVolume(volume constructionVolume, position DesignVector) map[DesignVector]bool {
	cells := make(map[DesignVector]bool, len(volume.cells))
	for _, p := range volume.cells {
		cells[DesignVector{p.X + position.X, p.Y + position.Y, p.Z + position.Z}] = true
	}
	return cells
}

// Resolve typed relative anchors before tiling. This computes positions only;
// it never clips overlaps, fills holes or declares a connection mechanically valid.
func resolveSceneAttachments(ctx context.Context, source SceneSpec, catalog PartCatalog) (SceneSpec, error) {
	scene := source
	scene.Objects = append([]SceneObject{}, source.Objects...)
	for i, object := range scene.Objects {
		modules := append([]Construction{}, object.Modules...)
		byID := map[string]int{}
		for n, m := range modules {
			byID[m.ID] = n
		}
		state := make([]int, len(modules))
		volumes := make([]constructionVolume, len(modules))
		world := make([]map[DesignVector]bool, len(modules))
		occupied, painted := map[DesignVector]bool{}, map[DesignVector]int{}
		for n, m := range modules {
			if m.Kind == "paint" && m.Attach != nil {
				m.Parent = m.Attach.Parent
			}
			m.Position = DesignVector{}
			v, err := moduleVolume(m, catalog)
			if err != nil {
				return scene, err
			}
			volumes[n] = v
		}
		var resolve func(int) error
		resolve = func(n int) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if state[n] == 2 {
				return nil
			}
			if state[n] == 1 {
				return designError("cyclic construction attachment: " + modules[n].ID)
			}
			state[n] = 1
			module := modules[n]
			a := module.Attach
			if a != nil {
				if module.Kind == "paint" {
					module.Parent = a.Parent
				}
				own := volumes[n].bounds
				size := DesignVector{own.max.X - own.min.X + 1, own.max.Y - own.min.Y + 1, own.max.Z - own.min.Z + 1}
				bottom := DesignVector{}
				if a.Anchor == "ground" && a.Parent == "" {
					bottom = a.Offset
				} else {
					parentIDs := a.Parents
					if len(parentIDs) == 0 {
						parentIDs = []string{a.Parent}
					}
					parent, ok := byID[parentIDs[0]]
					if !ok {
						return designError("unknown attachment parent " + a.Parent + " for " + module.ID)
					}
					if err := resolve(parent); err != nil {
						return err
					}
					bounds := volumes[parent].bounds
					pos := modules[parent].Position
					bounds.min = DesignVector{bounds.min.X + pos.X, bounds.min.Y + pos.Y, bounds.min.Z + pos.Z}
					bounds.max = DesignVector{bounds.max.X + pos.X, bounds.max.Y + pos.Y, bounds.max.Z + pos.Z}
					for _, id := range parentIDs[1:] {
						other, ok := byID[id]
						if !ok {
							return designError("unknown attachment parent " + id)
						}
						if err := resolve(other); err != nil {
							return err
						}
						b := volumes[other].bounds
						pos := modules[other].Position
						b.min = DesignVector{b.min.X + pos.X, b.min.Y + pos.Y, b.min.Z + pos.Z}
						b.max = DesignVector{b.max.X + pos.X, b.max.Y + pos.Y, b.max.Z + pos.Z}
						if b.max.Y != bounds.max.Y {
							return designError("top parents must end at the same height: " + module.ID)
						}
						bounds.min = DesignVector{min(bounds.min.X, b.min.X), min(bounds.min.Y, b.min.Y), min(bounds.min.Z, b.min.Z)}
						bounds.max = DesignVector{max(bounds.max.X, b.max.X), max(bounds.max.Y, b.max.Y), max(bounds.max.Z, b.max.Z)}
					}
					extent := DesignVector{bounds.max.X - bounds.min.X + 1, bounds.max.Y - bounds.min.Y + 1, bounds.max.Z - bounds.min.Z + 1}
					bottom = DesignVector{bounds.min.X + (extent.X-size.X)/2, bounds.max.Y + 1, bounds.min.Z + (extent.Z-size.Z)/2}
					switch a.Anchor {
					case "top":
						if module.Kind == "paint" {
							bottom.Y = bounds.max.Y - size.Y + 1
						}
					case "front", "back", "left", "right", "inside":
						if module.Kind != "paint" {
							return designError("side anchors require an embedded paint feature, not invented side connectors: " + module.ID)
						}
						bottom.Y = bounds.min.Y + max(0, (extent.Y-size.Y+5)/6)*3
						switch a.Anchor {
						case "front":
							bottom.Z = bounds.min.Z
						case "back":
							bottom.Z = bounds.max.Z - size.Z + 1
						case "left":
							bottom.X = bounds.min.X
						case "right":
							bottom.X = bounds.max.X - size.X + 1
						}
					default:
						return designError("unknown construction anchor " + a.Anchor)
					}
					bottom.X += a.Offset.X
					bottom.Y += a.Offset.Y
					bottom.Z += a.Offset.Z
					if size.X <= extent.X {
						bottom.X = max(bounds.min.X, min(bottom.X, bounds.max.X-size.X+1))
					}
					if size.Z <= extent.Z {
						bottom.Z = max(bounds.min.Z, min(bottom.Z, bounds.max.Z-size.Z+1))
					}
					if module.Kind == "paint" && (size.X > extent.X || size.Y > extent.Y || size.Z > extent.Z) {
						return designError("paint feature is larger than its parent: " + module.ID)
					}
					if module.Kind == "paint" {
						bottom.Y = max(bounds.min.Y, min(bottom.Y, bounds.max.Y-size.Y+1))
					}
					// Cache each construction's actual occupancy once. Search position
					// preferences without changing its dimensions, holes or colors.
					if module.Kind == "paint" || a.Anchor == "top" {
						type candidate struct {
							bottom   DesignVector
							distance int
						}
						candidates := []candidate{}
						minY, maxY := bottom.Y, bottom.Y
						if module.Kind == "paint" {
							minY, maxY = bounds.min.Y, bounds.max.Y-size.Y+1
						}
						minX, maxX, minZ, maxZ := bounds.min.X, bounds.max.X-size.X+1, bounds.min.Z, bounds.max.Z-size.Z+1
						if module.Kind != "paint" {
							minX, maxX = bounds.min.X-size.X+1, bounds.max.X
							minZ, maxZ = bounds.min.Z-size.Z+1, bounds.max.Z
						}
						for y := minY; y <= maxY; y++ {
							if module.Kind == "paint" && (y-bottom.Y)%3 != 0 {
								continue
							}
							for x := minX; x <= maxX; x++ {
								for z := minZ; z <= maxZ; z++ {
									distance := abs(x-bottom.X) + abs(z-bottom.Z) + abs(y-bottom.Y)
									if module.Kind == "paint" {
										switch a.Anchor {
										case "front", "back":
											distance += 100 * abs(z-bottom.Z)
										case "left", "right":
											distance += 100 * abs(x-bottom.X)
										case "top":
											distance += 100 * abs(y-bottom.Y)
										}
									}
									candidates = append(candidates, candidate{DesignVector{x, y, z}, distance})
								}
							}
						}
						sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].distance < candidates[j].distance })
						found := false
						for _, candidate := range candidates {
							if err := ctx.Err(); err != nil {
								return err
							}
							position := DesignVector{candidate.bottom.X - own.min.X, candidate.bottom.Y - own.min.Y, candidate.bottom.Z - own.min.Z}
							valid := true
							contacts := make([]bool, len(parentIDs))
							for index, p := range volumes[n].cells {
								if index%512 == 0 {
									if err := ctx.Err(); err != nil {
										return err
									}
								}
								q := DesignVector{p.X + position.X, p.Y + position.Y, p.Z + position.Z}
								if module.Kind == "paint" {
									_, alreadyPainted := painted[q]
									if !world[parent][q] || alreadyPainted {
										valid = false
										break
									}
									// A face anchor is a visibility requirement, not a
									// preference that can be satisfied by burying a mark.
									outward := DesignVector{}
									edge := false
									switch a.Anchor {
									case "front":
										edge, outward = p.Z == own.min.Z, DesignVector{Z: -1}
									case "back":
										edge, outward = p.Z == own.max.Z, DesignVector{Z: 1}
									case "left":
										edge, outward = p.X == own.min.X, DesignVector{X: -1}
									case "right":
										edge, outward = p.X == own.max.X, DesignVector{X: 1}
									case "top":
										edge, outward = p.Y == own.max.Y, DesignVector{Y: 1}
									}
									if edge && world[parent][DesignVector{q.X + outward.X, q.Y + outward.Y, q.Z + outward.Z}] {
										valid = false
										break
									}
									// Distinct marks of the same color on one parent must
									// remain distinguishable. Otherwise eyes and a nose
									// can merge into one stripe despite distinct IDs.
									for _, delta := range paintMarkNeighbors {
										other, exists := painted[DesignVector{q.X + delta.X, q.Y + delta.Y, q.Z + delta.Z}]
										if exists && modules[other].Color == module.Color && modules[other].Parent == module.Parent {
											valid = false
											break
										}
									}
									if !valid {
										break
									}
								} else {
									if occupied[q] {
										valid = false
										break
									}
									bearing := false
									for j, id := range parentIDs {
										if world[byID[id]][DesignVector{q.X, q.Y - 1, q.Z}] {
											contacts[j] = true
											bearing = true
										}
									}
									// A generated support assembly lands all of its pier
									// bottoms on the parent surface. One touching corner
									// cannot stand in for its distributed support grid.
									if (module.Kind == "platform" || module.Kind == "storey") && p.Y == own.min.Y && !bearing {
										valid = false
										break
									}
								}
							}
							if module.Kind != "paint" {
								for _, contact := range contacts {
									if !contact {
										valid = false
									}
								}
							}
							if valid {
								bottom = candidate.bottom
								found = true
								break
							}
						}
						if !found {
							if module.Kind == "paint" {
								return designError("no distinct exposed position for " + module.ID + " inside " + parentIDs[0] + "; keep face marks visible, preserve openings and leave a one-cell gap between separate marks of the same color, enlarge the parent or adjust patch dimensions/offsets")
							}
							return designError("no non-overlapping supported position for " + module.ID + " on " + parentIDs[0])
						}
					}
				}
				module.Position = DesignVector{bottom.X - own.min.X, bottom.Y - own.min.Y, bottom.Z - own.min.Z}
				if a.Anchor == "ground" && bottom.Y != 0 {
					return designError("ground attachment must stay at Y=0")
				}
			}
			modules[n] = module
			world[n] = translatedVolume(volumes[n], module.Position)
			for q := range world[n] {
				if module.Kind == "paint" {
					painted[q] = n
				} else {
					occupied[q] = true
				}
			}
			state[n] = 2
			return nil
		}
		for n := range modules {
			if err := resolve(n); err != nil {
				return scene, fmt.Errorf("%s/%s: %w", object.ID, modules[n].ID, err)
			}
		}
		scene.Objects[i].Modules = modules
	}
	return scene, nil
}
