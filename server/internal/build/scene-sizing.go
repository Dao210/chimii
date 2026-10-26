package build

import (
	"fmt"
	"strings"
)

// FitGeneratedScene selects minimum legal construction dimensions from a
// model's preferences. It applies only to generation/repair, never to the
// authoritative compiler or user-supplied API recipes. No feature, color or
// opening is removed; any resulting out-of-budget scene is still rejected.
func FitGeneratedScene(recipe AssemblyRecipe) AssemblyRecipe {
	if recipe.Version != 4 || recipe.Scene == nil {
		return recipe
	}
	scene := *recipe.Scene
	scene.Objects = append([]SceneObject{}, scene.Objects...)
	changes := []string{}
	for i, o := range scene.Objects {
		o.Modules = append([]Construction{}, o.Modules...)
		for n, m := range o.Modules {
			before := m.Size
			if m.Kind != "shape" && (m.Size.X < 1 || m.Size.X > 32 || m.Size.Z < 1 || m.Size.Z > 32 || m.Size.Y < 1 || m.Size.Y > 96) {
				// Out-of-range input is an error, not a sizing preference.
				continue
			}
			switch m.Kind {
			case "wheelbase":
				m.Size.X = max(4, ((m.Size.X+1)/2)*2)
				m.Size.Z = max(4, ((m.Size.Z+1)/2)*2)
				m.Size.Y = 5
				if m.Attach != nil && m.Attach.Anchor == "ground" {
					m.Position.Y = 0
				}
			case "platform":
				m.Size.X, m.Size.Z = max(4, m.Size.X), max(4, m.Size.Z)
				m.Size.Y = max(9, ((m.Size.Y+2)/3)*3)
			case "taper", "roof":
				if m.Size.Y > 0 {
					m.Size.Y = ((m.Size.Y + 2) / 3) * 3
				}
				width := 2 + 2*(m.Size.Y/3-1)
				m.Size.X = max(m.Size.X, width)
				if m.Kind == "taper" {
					m.Size.Z = max(m.Size.Z, width)
				}
			case "stairs":
				if m.Size.Z > 0 {
					m.Size.Z = ((m.Size.Z + 1) / 2) * 2
					m.Size.Y = 3 * (m.Size.Z / 2)
				}
			case "wall", "frame", "storey":
				if m.Kind == "storey" {
					m.Size.X, m.Size.Z = max(6, m.Size.X), max(6, m.Size.Z)
					m.Size.Y = max(12, ((m.Size.Y+2)/3)*3)
					m.CeilingOpenings = append([]DeckOpening{}, m.CeilingOpenings...)
					for _, opening := range m.CeilingOpenings {
						if opening.X >= 2 && opening.Z >= 2 && opening.Width > 0 && opening.Depth > 0 {
							m.Size.X, m.Size.Z = max(m.Size.X, opening.X+opening.Width+2), max(m.Size.Z, opening.Z+opening.Depth+2)
						}
					}
				}
				m.Openings = append([]WallOpening{}, m.Openings...)
				for j, opening := range m.Openings {
					if opening.X < 0 || opening.Y < 0 || opening.Width < 1 || opening.Height < 1 {
						continue
					}
					old := opening.X
					opening.X = max(2, opening.X)
					if old != opening.X {
						changes = append(changes, fmt.Sprintf("%s/%s opening %d: pier offset %d -> %d", o.ID, m.ID, j, old, opening.X))
					}
					length := opening.X + opening.Width + 2
					if constructionOpeningAlongZ(m, opening) {
						m.Size.Z = max(m.Size.Z, length)
					} else {
						m.Size.X = max(m.Size.X, length)
					}
					lintel := 3
					if m.Kind == "storey" {
						lintel = 9
					}
					m.Size.Y = max(m.Size.Y, opening.Y+opening.Height+lintel)
					if m.Kind == "storey" {
						m.Size.Y = ((m.Size.Y + 2) / 3) * 3
					}
					m.Openings[j] = opening
				}
			}
			if m.Size != before {
				changes = append(changes, fmt.Sprintf("%s/%s size (%d,%d,%d) -> (%d,%d,%d)", o.ID, m.ID, before.X, before.Y, before.Z, m.Size.X, m.Size.Y, m.Size.Z))
			}
			o.Modules[n] = m
		}
		scene.Objects[i] = o
	}
	recipe.Scene = &scene
	if len(changes) > 0 {
		metadata := map[string]string{}
		for k, v := range recipe.Metadata {
			metadata[k] = v
		}
		metadata["construction_sizing"] = strings.Join(changes, "; ")
		recipe.Metadata = metadata
	}
	return recipe
}
