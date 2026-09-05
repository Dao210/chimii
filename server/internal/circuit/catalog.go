package circuit

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog.json
var catalogJSON []byte

// StarterCatalog returns a fresh snapshot: callers cannot mutate the shared
// reference catalogue or another request's validation authority.
func StarterCatalog() Catalog {
	var c Catalog
	if err := json.Unmarshal(catalogJSON, &c); err != nil {
		panic(err)
	}
	return c
}

func (c Catalog) Part(id string) (Part, bool) {
	for _, p := range c.Parts {
		if p.ID == id {
			return p, true
		}
	}
	return Part{}, false
}

func (c Catalog) Project(id string) (Project, bool) {
	for _, p := range c.Projects {
		if p.ID == id {
			return p, true
		}
	}
	return Project{}, false
}

func (c Catalog) Inventory() map[string]int {
	i := make(map[string]int, len(c.Parts))
	for _, p := range c.Parts {
		i[p.ID] = p.Quantity
	}
	return i
}

func Transform(p Placement, x, y int) Point {
	switch p.Rotation {
	case 90:
		x, y = -y, x
	case 180:
		x, y = -x, -y
	case 270:
		x, y = y, -x
	}
	return Point{p.X + x, p.Y + y}
}
