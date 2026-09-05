package circuit

import (
	_ "embed"
	"encoding/json"
)

//go:embed catalog.json
var catalogJSON []byte

//go:embed boson.json
var bosonJSON []byte

// Catalogues share the document lifecycle while each connection system owns
// its compiler rules. Similar-looking kits are never aliases.
func Catalogs() []Catalog {
	var boson Catalog
	if err := json.Unmarshal(bosonJSON, &boson); err != nil {
		panic(err)
	}
	return []Catalog{StarterCatalog(), boson}
}

func FindCatalog(kitID string) (Catalog, bool) {
	for _, c := range Catalogs() {
		if c.KitID == kitID {
			return c, true
		}
	}
	return Catalog{}, false
}

// StarterCatalog returns a fresh snapshot: callers cannot mutate the shared
// reference catalogue or another request's validation authority.
func StarterCatalog() Catalog {
	var c Catalog
	if err := json.Unmarshal(catalogJSON, &c); err != nil {
		panic(err)
	}
	c.Hardware = &HardwareProfile{
		Manufacturer: "Elenco", Model: "SC-500", CheckedOn: "2026-09-05",
		Notes: Text{EN: "Only the supported project subset is listed. Confirm SC-500 part markings and stock with the supplier before buying.", ZH: "这里只列出支持项目所需的元件子集，采购前向卖家确认 SC-500 元件编号和现货。"},
		PurchaseLinks: []PurchaseLink{
			{Label: Text{EN: "Official international retailers", ZH: "官方国际经销渠道"}, URL: "https://elenco.com/stores-international/"},
			{Label: Text{EN: "Replacement parts", ZH: "官方补件渠道"}, URL: "https://elenco.com/replacement-parts/"},
		},
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
